package dag

import (
	"context"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/iykyk-syn/unison/bapl"
	"github.com/iykyk-syn/unison/crypto"
	"github.com/iykyk-syn/unison/crypto/ed25519"
	"github.com/iykyk-syn/unison/dag/block"
	"github.com/iykyk-syn/unison/dag/quorum"
	"github.com/iykyk-syn/unison/rebro"
)

// EthereumClient defines the methods used from the Ethereum client.
type EthereumClient interface {
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

// RealEthereumClient is a wrapper around the actual ethclient.Client that implements the EthereumClient interface.
type RealEthereumClient struct {
	client *ethclient.Client
}

func (r *RealEthereumClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	return r.client.CallContract(ctx, msg, blockNumber)
}

// NewRealEthereumClient creates a new instance of RealEthereumClient.
func NewRealEthereumClient(url string) (*RealEthereumClient, error) {
	client, err := ethclient.Dial(url)
	if err != nil {
		return nil, err
	}
	return &RealEthereumClient{client: client}, nil
}

type IncludersFn func(round uint64) (*quorum.Includers, error)

// Chain produces everlasting DAG chain of blocks broadcasting them over reliable broadcast.
type Chain struct {
	broadcaster    rebro.Broadcaster
	batchPool      bapl.BatchPool
	includers      *quorum.Includers
	signerID       crypto.PubKey
	stakingConfig  *StakingConfig
	height         uint64
	lastCerts      []rebro.Certificate
	log            *slog.Logger
	cancel         context.CancelFunc
	ethClient      EthereumClient
	stakingABI     abi.ABI
	stakingAddress common.Address
}

const stakingABIJSON = `[
	{
		"constant": true,
		"inputs": [],
		"name": "getStakers",
		"outputs": [
			{
				"name": "",
				"type": "address[]"
			},
			{
				"name": "",
				"type": "bytes32[]"
			}
		],
		"payable": false,
		"stateMutability": "view",
		"type": "function"
	}
]`

func NewChainWithStakingConfig(
	broadcaster rebro.Broadcaster,
	pool bapl.BatchPool,
	stakingConfig *StakingConfig,
	signerID crypto.PubKey,
) *Chain {
	ethClient, err := NewRealEthereumClient(stakingConfig.EthRpcUrl)
	if err != nil {
		panic(err)
	}

	stakingABI, err := abi.JSON(strings.NewReader(stakingABIJSON))
	if err != nil {
		panic(err)
	}

	chain := &Chain{
		broadcaster:    broadcaster,
		batchPool:      pool,
		signerID:       signerID,
		stakingConfig:  stakingConfig,
		height:         1, // must start from 1
		log:            slog.With("module", "dagger"),
		ethClient:      ethClient,
		stakingABI:     stakingABI,
		stakingAddress: common.HexToAddress(stakingConfig.StakingAddress),
	}
	err = chain.updateIncluders()
	if err != nil {
		panic(err)
	}
	return chain
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
// * collect block hashes from last height as parent hashes
// * cleanup batches committed in blocks from last height
// * prepare the new uncommitted batches
// * create a block from the batches and the parents hashes;
// * propagate the block and wait until quorum is reached;
func (c *Chain) startRound(ctx context.Context) error {
	// Check if it's time to update includers
	if c.height%c.stakingConfig.BlocksPerEpoch == 0 {
		err := c.updateIncluders()
		if err != nil {
			return fmt.Errorf("failed to update includers: %w", err)
		}
	}

	parents := make([][]byte, len(c.lastCerts))
	for i, cert := range c.lastCerts {
		parents[i] = cert.Message().ID.Hash()

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
		return fmt.Errorf("can't get batches for the new height:%w", err)
	}

	// TODO: certificate signatures should be the part of the block.
	blk := block.NewBlock(c.height, c.signerID.Bytes(), newBatches, parents)
	blk.Hash() // TODO: Compute in constructor
	data, err := blk.MarshalBinary()
	if err != nil {
		return err
	}

	includers := c.includers
	if includers == nil {
		return fmt.Errorf("includers list is not initialized")
	}

	now := time.Now()
	msg := rebro.Message{ID: blk.ID(), Data: data}
	qrm := quorum.NewQuorum(includers)
	err = c.broadcaster.Broadcast(ctx, msg, qrm)
	if err != nil {
		return err
	}
	c.log.InfoContext(ctx, "finished round", "height", c.height, "batches", len(newBatches), "parents", len(parents), "time", time.Since(now))

	c.lastCerts = qrm.List()
	c.height++
	return nil
}

const defaultStake = 1000

func (c *Chain) updateIncluders() error {
	callData, err := c.stakingABI.Pack("getStakers")
	if err != nil {
		return fmt.Errorf("error packing call data: %w", err)
	}

	msg := ethereum.CallMsg{
		To:   &c.stakingAddress,
		Data: callData,
	}

	result, err := c.ethClient.CallContract(context.Background(), msg, nil)
	if err != nil {
		return fmt.Errorf("error calling contract: %w", err)
	}

	var (
		addresses []common.Address
		pubKeys   []common.Hash
	)
	err = c.stakingABI.UnpackIntoInterface(&[]interface{}{&addresses, &pubKeys}, "getStakers", result)
	if err != nil {
		return fmt.Errorf("error unpacking result: %w", err)
	}

	includers := make([]*quorum.Includer, len(pubKeys))
	for i, pubKey := range pubKeys {
		// Convert common.Hash to crypto.PubKey using BytesToPubKey
		cryptoPubKey, err := ed25519.BytesToPubKey(pubKey.Bytes())
		if err != nil {
			return fmt.Errorf("error converting public key bytes to crypto.PubKey: %w", err)
		}
		includers[i] = quorum.NewIncluder(cryptoPubKey, defaultStake)
	}
	c.includers = quorum.NewIncludersSet(includers)
	return nil
}

// StakingConfig holds the configuration for the staking contract.
type StakingConfig struct {
	EthRpcUrl      string
	StakingAddress string
	BlocksPerEpoch uint64
}
