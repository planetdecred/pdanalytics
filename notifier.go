package main

import (
	"context"
	"reflect"
	"runtime"
	"sync"
	"time"

	"github.com/decred/dcrd/chaincfg/chainhash"
	"github.com/decred/dcrd/rpcclient/v5"
	"github.com/decred/dcrd/wire"
)

// SyncHandlerDeadline is a hard deadline for handlers to finish handling before
// an error is logged.
const SyncHandlerDeadline = time.Minute * 5

// BlockHandler is a function that will be called when dcrd reports a new block.
type BlockHandler func(*wire.BlockHeader) error

type notifier struct {
	ctx  context.Context
	node *rpcclient.Client
	// The anyQ sequences all dcrd notification in the order they are received.
	anyQ     chan interface{}
	block    [][]BlockHandler
	previous struct {
		hash   chainhash.Hash
		height uint32
	}
}

// NewNotifier is the constructor for a Notifier.
func NewNotifier(ctx context.Context) *notifier {
	return &notifier{
		ctx: ctx,
		// anyQ can cause deadlocks if it gets full. All mempool transactions pass
		// through here, so the size should stay pretty big to accommodate for the
		// inevitable explosive growth of the network.
		anyQ:  make(chan interface{}, 1024),
		block: make([][]BlockHandler, 0),
	}
}

// Listen must be called once, but only after all handlers are registered.
func (notifier *notifier) Listen(dcrdClient *rpcclient.Client) *ContextualError {
	// Register for block connection and chain reorg notifications.
	notifier.node = dcrdClient

	var err error
	if err = dcrdClient.NotifyBlocks(); err != nil {
		return newContextualError("block notification "+
			"registration failed", err)
	}

	// Register for tx accepted into mempool ntfns
	if err = dcrdClient.NotifyNewTransactions(true); err != nil {
		return newContextualError("new transaction verbose notification registration failed", err)
	}

	if err = dcrdClient.NotifyWinningTickets(); err != nil {
		return newContextualError("winning ticket "+
			"notification registration failed", err)
	}

	go notifier.superQueue()
	return nil
}

// DcrdHandlers creates a set of handlers to be passed to the dcrd
// rpcclient.Client as a parameter of its constructor.
func (notifier *notifier) DcrdHandlers() *rpcclient.NotificationHandlers {
	return &rpcclient.NotificationHandlers{
		OnBlockConnected:    notifier.onBlockConnected,
		OnBlockDisconnected: notifier.onBlockDisconnected,
	}
}

// RegisterBlockHandlerGroup adds a group of block handlers. Groups are run
// sequentially in the order they are registered, but the handlers within the
// group are run asynchronously. Handlers registered with
// RegisterBlockHandlerGroup are FIFO'd together with handlers registered with
// RegisterBlockHandlerLiteGroup.
func (notifier *notifier) RegisterBlockHandlerGroup(handlers ...BlockHandler) {
	notifier.block = append(notifier.block, handlers)
}

// SetPreviousBlock modifies the height and hash of the best block. This data is
// required to avoid connecting new blocks that are not next in the chain. It is
// only necessary to call SetPreviousBlock if blocks are connected or
// disconnected by a mechanism other than (*Notifier).processBlock, which
// keeps this data up-to-date. For example, signalReorg will use
// SetPreviousBlock after the reorg is complete.
func (notifier *notifier) SetPreviousBlock(prevHash chainhash.Hash, prevHeight uint32) {
	notifier.previous.hash = prevHash
	notifier.previous.height = prevHeight
}

func functionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

// superQueue should be run as a goroutine. The dcrd-registered block and reorg
// handlers should perform any pre-processing and type conversion and then
// deposit the payload into the anyQ channel.
func (notifier *notifier) superQueue() {
out:
	for {
		select {
		case rawMsg := <-notifier.anyQ:
			// Do not allow new blocks to process while running reorg. Only allow
			// them to be processed after this reorg completes.
			switch msg := rawMsg.(type) {
			case *wire.BlockHeader:
				// Process the new block.
				log.Infof("superQueue: Processing new block %v (height %d).", msg.BlockHash(), msg.Height)
				notifier.processBlock(msg)
			default:
				log.Warn("unknown/unhandled message type in superQueue: %T", rawMsg)
			}
		case <-notifier.ctx.Done():
			break out
		}
	}
}

// processBlock calls the BlockHandler/BlockHandlerLite groups one at a time in
// the order that they were registered.
func (notifier *notifier) processBlock(bh *wire.BlockHeader) {
	hash := bh.BlockHash()
	height := bh.Height
	prev := notifier.previous

	// Ensure that the received block (bh.hash, bh.height) connects to the
	// previously connected block (q.prevHash, q.prevHeight).
	if bh.PrevBlock != prev.hash {
		log.Infof("Received block at %d (%v) does not connect to %d (%v). "+
			"This is normal before reorganization.",
			height, hash, prev.height, prev.hash)
		return
	}

	start := time.Now()
	for _, handlers := range notifier.block {
		wg := new(sync.WaitGroup)
		for _, h := range handlers {
			wg.Add(1)
			go func(h BlockHandler) {
				tStart := time.Now()
				defer wg.Done()
				defer log.Tracef("Notifier: BlockHandler %s completed in %v",
					functionName(h), time.Since(tStart))
				if err := h(bh); err != nil {
					log.Errorf("block handler failed: %v", err)
					return
				}
			}(h)
		}
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.NewTimer(SyncHandlerDeadline).C:
			log.Errorf("at least 1 block handler has not completed before the deadline")
			return
		}
	}
	log.Debugf("handlers of Notifier.processBlock() completed in %v", time.Since(start))

	// Record this block as the best block connected by the collectionQueue.
	notifier.SetPreviousBlock(hash, height)
}

// rpcclient.NotificationHandlers.OnBlockConnected
func (notifier *notifier) onBlockConnected(blockHeaderSerialized []byte, _ [][]byte) {
	blockHeader := new(wire.BlockHeader)
	err := blockHeader.FromBytes(blockHeaderSerialized)
	if err != nil {
		log.Error("Failed to deserialize blockHeader in new block notification: "+
			"%v", err)
		return
	}
	height := int32(blockHeader.Height)
	hash := blockHeader.BlockHash()
	prevHash := blockHeader.PrevBlock // to ensure this is the next block

	log.Debugf("OnBlockConnected: %d / %v (previous: %v)", height, hash, prevHash)

	notifier.anyQ <- blockHeader
}

// rpcclient.NotificationHandlers.OnBlockDisconnected
func (notifier *notifier) onBlockDisconnected(blockHeaderSerialized []byte) {
	blockHeader := new(wire.BlockHeader)
	err := blockHeader.FromBytes(blockHeaderSerialized)
	if err != nil {
		log.Error("Failed to deserialize blockHeader in block disconnect notification: "+
			"%v", err)
		return
	}
	height := int32(blockHeader.Height)
	hash := blockHeader.BlockHash()

	log.Debugf("OnBlockDisconnected: %d / %v", height, hash)
}
