package propagation

import (
	"context"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	chainjson "github.com/decred/dcrd/rpc/jsonrpc/types/v2"
	"github.com/decred/dcrd/wire"
	"github.com/planetdecred/pdanalytics/dcrd"
	"github.com/planetdecred/pdanalytics/web"
)

func New(ctx context.Context, client *dcrd.Dcrd, dataStore Store, externalDBs map[string]Store,
	webServer *web.Server) (*propagation, error) {

	var externalDBNames []string
	for n := range externalDBs {
		externalDBNames = append(externalDBNames, n)
	}

	prop := &propagation{
		ctx:             ctx,
		dataStore:       dataStore,
		externalDBs:     externalDBs,
		server:          webServer,
		ticketInds:      make(dcrd.BlockValidatorIndex),
		client:          client,
		externalDBNames: externalDBNames,
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
		BlockReceiveTime:  web.NowUTC(),
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
	if !prop.syncIsDone {
		return nil
	}
	receiveTime := web.NowUTC()

	msgTx, err := dcrd.MsgTxFromHex(txDetails.Hex)
	if err != nil {
		log.Errorf("Failed to decode transaction hex: %v", err)
		return err
	}

	if txType := dcrd.DetermineTxTypeString(msgTx); txType != "Vote" {
		return nil
	}

	var voteInfo *dcrd.VoteInfo
	validation, version, bits, choices, err := dcrd.SSGenVoteChoices(msgTx, prop.client.Params)
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

	prop.ticketIndsMutex.Lock()
	voteInfo.SetTicketIndex(prop.ticketInds)
	prop.ticketIndsMutex.Unlock()

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
		targetedBlock, err = prop.client.Rpc.GetBlock(hash)
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

	if err = prop.dataStore.SaveVote(prop.ctx, vote); err != nil {
		log.Error(err)
	}

	if err = prop.dataStore.UpdateVoteTimeDeviationData(prop.ctx); err != nil {
		log.Errorf("Error in vote receive time deviation data update, %s", err.Error())
	}
	return nil
}
