package dag

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/iykyk-syn/unison/bapl"
	"github.com/iykyk-syn/unison/crypto"
	"github.com/iykyk-syn/unison/dag/block"
	"github.com/iykyk-syn/unison/dag/quorum"
	"github.com/iykyk-syn/unison/rebro"
)

type IncludersFn func(round uint64) (*quorum.Includers, error)

// Chain produces everlasting DAG chain of blocks broadcasting them over reliable broadcast.
type Chain struct {
	broadcaster rebro.Broadcaster
	batchPool   bapl.BatchPool
	includers   IncludersFn
	signerID    crypto.PubKey

	round      uint64
	lastQuorum rebro.QuorumCertificate

	log    *slog.Logger
	cancel context.CancelFunc
}

func NewChain(
	broadcaster rebro.Broadcaster,
	pool bapl.BatchPool,
	includers IncludersFn,
	signerID crypto.PubKey,
) *Chain {
	return &Chain{
		broadcaster: broadcaster,
		batchPool:   pool,
		includers:   includers,
		signerID:    signerID,
		round:       1, // must start from 1
		log:         slog.With("module", "dagger"),
	}
}

func (c *Chain) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	go c.run(ctx)
	c.log.Debug("started")
	return
}

func (c *Chain) Stop() {
	c.cancel()
}

// run is indefinitely producing new blocks and broadcasts them across the network
func (c *Chain) run(ctx context.Context) {
	for ctx.Err() == nil {
		err := c.startRound(ctx)
		if err != nil {
			c.log.ErrorContext(ctx, "executing round", "reason", err)
			// temporary and hacky solution.
			// TODO: remove this in favour of better approach
			time.Sleep(time.Second * 3)
		}
	}
}

// startRound assembles a new block and broadcasts it across the network.
//
// assembling stages:
// * collect block hashes from last round as parent hashes
// * cleanup batches commited in blocks from last round
// * prepare the new uncommited batches
// * create a block from the batches and the parents hashes;
// * propagate the block and wait until quorum is reached;
func (c *Chain) startRound(ctx context.Context) error {
	certs := c.lastCertificates()
	parents := make([][]byte, len(certs))
	for i, cert := range certs {
		parents[i] = certs[i].Message().ID.Hash()

		var blk block.Block
		err := blk.UnmarshalBinary(cert.Message().Data)
		if err != nil {
			panic(err)
		}

		for _, batchHash := range blk.Batches() {
			err := c.batchPool.Delete(ctx, batchHash)
			if err != nil {
				c.log.WarnContext(ctx, "can't delete a batch", "err", err)
			}
		}
	}

	newBatches, err := c.batchPool.ListBySigner(ctx, c.signerID.Bytes())
	if err != nil {
		return fmt.Errorf("can't get batches for the new round:%w", err)
	}

	// TODO: certificate signatures should be the part of the block.
	blk := block.NewBlock(c.round, c.signerID.Bytes(), newBatches, parents)
	blk.Hash() // TODO: Compute in constructor
	data, err := blk.MarshalBinary()
	if err != nil {
		return err
	}

	includers, err := c.includers(c.round)
	if err != nil {
		return err
	}

	msg := rebro.Message{ID: blk.ID(), Data: data}
	qrm := quorum.NewQuorum(includers)
	err = c.broadcaster.Broadcast(ctx, msg, qrm)
	if err != nil {
		return err
	}
	c.log.InfoContext(ctx, "finished round", "round", c.round, "batches", len(newBatches), "parents", len(parents))

	c.round++
	c.lastQuorum = qrm
	return nil
}

func (c *Chain) lastCertificates() []rebro.Certificate {
	if c.lastQuorum == nil {
		return nil
	}

	return c.lastQuorum.List()
}
