package propagation

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v2"
	"github.com/decred/dcrd/wire"
	"github.com/planetdecred/dcrextdata/app/helpers"
	"github.com/planetdecred/pdanalytics/datasync"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

func New(ctx context.Context, client *dcrd.Dcrd, dataStore store,
	webServer *web.Server, syncCoordinator *datasync.SyncCoordinator) (*propagation, error) {

	prop := &propagation{
		ctx:        ctx,
		dataStore:  dataStore,
		server:     webServer,
		ticketInds: make(dcrd.BlockValidatorIndex),
		client:     client,
	}

	tmpls := []string{"propagation"}

	for _, name := range tmpls {
		if err := prop.server.Templates.AddTemplate(name); err != nil {
			log.Errorf("Unable to create new html template: %v", err)
			return nil, err
		}
	}

	prop.server.AddMenuItem(web.MenuItem{
		Href:      "/propagation",
		HyperText: "Propagation",
		Attributes: map[string]string{
			"class": "menu-item",
			"title": "Block Propagation",
		},
	})

	prop.server.AddRoute("/propagation", web.GET, prop.propagationPage)
	prop.server.AddRoute("/getpropagationdata", web.GET, prop.getPropagationData)
	prop.server.AddRoute("/getblocks", web.GET, prop.getBlocks)
	prop.server.AddRoute("/getvotes", web.GET, prop.getVotes)
	prop.server.AddRoute("/getvotebyblock", web.GET, prop.getVoteByBlock)
	prop.server.AddRoute("/api/charts/propagation/{chartDataType}", web.GET, prop.chart, chartDataTypeCtx)

	prop.client.Notif.RegisterBlockHandlerGroup(prop.ConnectBlock)
	prop.client.Notif.RegisterTxHandlerGroup(prop.TxReceived)

	prop.RegisterSyncer(syncCoordinator)

	return prop, nil
}

func (prop *propagation) ConnectBlock(blockHeader *wire.BlockHeader) error {
	if !prop.syncIsDone {
		done, err := prop.client.IsSynced()
		if err != nil {
			log.Infof("Unable to determine the sync status of dcrd, %v", err)
			return err
		}
		if !done {
			log.Infof("Received a staled block height %d, block dropped", blockHeader.Height)
			return nil
		}
		prop.syncIsDone = true
	}

	block := Block{
		BlockInternalTime: blockHeader.Timestamp.UTC(),
		BlockReceiveTime:  helpers.NowUTC(),
		BlockHash:         blockHeader.BlockHash().String(),
		BlockHeight:       blockHeader.Height,
	}
	if err := prop.dataStore.SaveBlock(prop.ctx, block); err != nil {
		log.Error(err)
		return err
	}
	if err := prop.dataStore.UpdateBlockBinData(prop.ctx); err != nil {
		log.Errorf("Error in block bin data update, %s", err.Error())
		return err
	}
	return nil
}

func (prop *propagation) TxReceived(txDetails *chainjson.TxRawResult) error {
	if !c.syncIsDone {
		return nil
	}
	receiveTime := helpers.NowUTC()

	msgTx, err := dcrd.MsgTxFromHex(txDetails.Hex)
	if err != nil {
		log.Errorf("Failed to decode transaction hex: %v", err)
		return err
	}

	if txType := dcrd.DetermineTxTypeString(msgTx); txType != "Vote" {
		return nil
	}

	var voteInfo *dcrd.VoteInfo
	validation, version, bits, choices, err := dcrd.SSGenVoteChoices(msgTx, c.client.Params)
	if err != nil {
		log.Errorf("Error in getting vote choice: %s", err.Error())
		return err
	}

	voteInfo = &dcrd.VoteInfo{
		Validation: dcrd.BlockValidation{
			Hash:     validation.Hash,
			Height:   validation.Height,
			Validity: validation.Validity,
		},
		Version:     version,
		Bits:        bits,
		Choices:     choices,
		TicketSpent: msgTx.TxIn[1].PreviousOutPoint.Hash.String(),
	}

	c.ticketIndsMutex.Lock()
	voteInfo.SetTicketIndex(c.ticketInds)
	c.ticketIndsMutex.Unlock()

	vote := Vote{
		ReceiveTime: receiveTime,
		VotingOn:    validation.Height,
		Hash:        txDetails.Txid,
		ValidatorId: voteInfo.MempoolTicketIndex,
	}

	if voteInfo.Validation.Validity {
		vote.Validity = "Valid"
	} else {
		vote.Validity = "Invalid"
	}

	var retries = 3
	var targetedBlock *wire.MsgBlock

	// try to get the block from the blockchain until the number of retries has elapsed
	for i := 0; i <= retries; i++ {
		hash, _ := chainhash.NewHashFromStr(validation.Hash)
		targetedBlock, err = c.client.Rpc.GetBlock(hash)
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	// err is ignored since the vote will be updated when the block becomes available
	if targetedBlock != nil {
		vote.TargetedBlockTime = targetedBlock.Header.Timestamp.UTC()
		vote.BlockHash = targetedBlock.Header.BlockHash().String()
	}

	if err = c.dataStore.SaveVote(c.ctx, vote); err != nil {
		log.Error(err)
	}

	if err = c.dataStore.UpdateVoteTimeDeviationData(c.ctx); err != nil {
		log.Errorf("Error in vote receive time deviation data update, %s", err.Error())
	}
	return nil
}

func (prop *propagation) RegisterSyncer(syncCoordinator *datasync.SyncCoordinator) {
	prop.registerBlockSyncer(syncCoordinator)
	prop.registerVoteSyncer(syncCoordinator)
}

func (prop *propagation) registerBlockSyncer(syncCoordinator *datasync.SyncCoordinator) {
	syncCoordinator.AddSyncer(prop.dataStore.BlockTableName(), datasync.Syncer{
		LastEntry: func(ctx context.Context, db datasync.Store) (string, error) {
			var lastHeight int64
			err := db.LastEntry(ctx, prop.dataStore.BlockTableName(), &lastHeight)
			if err != nil && err != sql.ErrNoRows {
				return "0", fmt.Errorf("error in fetching last block height, %s", err.Error())
			}
			return strconv.FormatInt(lastHeight, 10), nil
		},
		Collect: func(ctx context.Context, url string) (result *datasync.Result, err error) {
			result = new(datasync.Result)
			result.Records = []Block{}
			err = helpers.GetResponse(ctx, &http.Client{Timeout: 10 * time.Second}, url, result)
			return
		},
		Retrieve: func(ctx context.Context, last string, skip, take int) (result *datasync.Result, err error) {
			blockHeight, _ := strconv.ParseInt(last, 10, 64)
			result = new(datasync.Result)
			blocks, totalCount, err := prop.dataStore.FetchBlockForSync(ctx, blockHeight, skip, take)
			if err != nil {
				result.Message = err.Error()
				return
			}
			result.Records = blocks
			result.TotalCount = totalCount
			result.Success = true
			return
		},
		Append: func(ctx context.Context, store datasync.Store, data interface{}) {
			mappedData := data.([]interface{})
			var blocks []Block
			for _, item := range mappedData {
				var block Block
				err := datasync.DecodeSyncObj(item, &block)
				if err != nil {
					log.Errorf("Error in decoding the received block data, %s", err.Error())
					return
				}
				blocks = append(blocks, block)
			}

			for _, block := range blocks {
				err := store.SaveFromSync(ctx, block)
				if err != nil {
					log.Errorf("Error while appending block synced data, %s", err.Error())
				}
			}
			// update propagation data
			if err := store.ProcessEntries(ctx); err != nil {
				log.Errorf("Error in initial propagation data update, %s", err.Error())
			}
		},
	})
}

func (prop *propagation) registerVoteSyncer(syncCoordinator *datasync.SyncCoordinator) {
	syncCoordinator.AddSyncer(prop.dataStore.VoteTableName(), datasync.Syncer{
		LastEntry: func(ctx context.Context, db datasync.Store) (string, error) {
			var receiveTime time.Time
			err := db.LastEntry(ctx, prop.dataStore.VoteTableName(), &receiveTime)
			if err != nil && err != sql.ErrNoRows {
				return "0", fmt.Errorf("error in fetching last vote receive time, %s", err.Error())
			}
			return strconv.FormatInt(receiveTime.Unix(), 10), nil
		},
		Collect: func(ctx context.Context, url string) (result *datasync.Result, err error) {
			result = new(datasync.Result)
			result.Records = []Vote{}
			err = helpers.GetResponse(ctx, &http.Client{Timeout: 10 * time.Second}, url, result)
			return
		},
		Retrieve: func(ctx context.Context, last string, skip, take int) (result *datasync.Result, err error) {
			unixDate, _ := strconv.ParseInt(last, 10, 64)
			result = new(datasync.Result)
			votes, totalCount, err := prop.dataStore.FetchVoteForSync(ctx, helpers.UnixTime(unixDate), skip, take)
			if err != nil {
				result.Message = err.Error()
				return
			}
			fmt.Println("Total count", totalCount)
			result.Records = votes
			result.TotalCount = totalCount
			result.Success = true
			return
		},
		Append: func(ctx context.Context, store datasync.Store, data interface{}) { //todo: should return an error
			mappedData := data.([]interface{})
			var votes []Vote
			for _, item := range mappedData {
				var vote Vote
				err := datasync.DecodeSyncObj(item, &vote)
				if err != nil {
					log.Errorf("Error in decoding the received vote data, %s", err.Error())
					return
				}
				votes = append(votes, vote)
			}

			for _, vote := range votes {
				err := store.SaveVoteFromSync(ctx, vote)
				if err != nil {
					log.Errorf("Error while appending vote synced data, %s", err.Error())
				}
			}
		},
	})
}
