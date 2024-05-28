package dag

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/iykyk-syn/unison/bapl"
	"github.com/iykyk-syn/unison/crypto"
	"github.com/iykyk-syn/unison/crypto/ed25519"
	"github.com/iykyk-syn/unison/rebro"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/host"
	mocknet "github.com/libp2p/go-libp2p/p2p/net/mock"
)

type MockBroadcaster struct {
	broadcastFn func(ctx context.Context, msg rebro.Message, qc rebro.QuorumCertificate) error
}

func (m *MockBroadcaster) Broadcast(ctx context.Context, msg rebro.Message, qc rebro.QuorumCertificate) error {
	if m.broadcastFn != nil {
		return m.broadcastFn(ctx, msg, qc)
	}
	return nil
}

type MockBatchPool struct {
	pushFn         func(ctx context.Context, batch *bapl.Batch) error
	pullFn         func(ctx context.Context, hash []byte) (*bapl.Batch, error)
	listBySignerFn func(ctx context.Context, signer []byte) ([]*bapl.Batch, error)
	deleteFn       func(ctx context.Context, hash []byte) error
	sizeFn         func(ctx context.Context) (int, error)
}

func (m *MockBatchPool) Push(ctx context.Context, batch *bapl.Batch) error {
	if m.pushFn != nil {
		return m.pushFn(ctx, batch)
	}
	return nil
}

func (m *MockBatchPool) Pull(ctx context.Context, hash []byte) (*bapl.Batch, error) {
	if m.pullFn != nil {
		return m.pullFn(ctx, hash)
	}
	return nil, nil
}

func (m *MockBatchPool) ListBySigner(ctx context.Context, signer []byte) ([]*bapl.Batch, error) {
	if m.listBySignerFn != nil {
		return m.listBySignerFn(ctx, signer)
	}
	return nil, nil
}

func (m *MockBatchPool) Delete(ctx context.Context, hash []byte) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, hash)
	}
	return nil
}

func (m *MockBatchPool) Size(ctx context.Context) (int, error) {
	if m.sizeFn != nil {
		return m.sizeFn(ctx)
	}
	return 0, nil
}

type MockEthereumClient struct {
	callContractFn func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func (m *MockEthereumClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if m.callContractFn != nil {
		return m.callContractFn(ctx, msg, blockNumber)
	}
	return nil, nil
}

func TestChain_UpdateIncluders_HaltsOnFailure(t *testing.T) {
	stakingABI, err := abi.JSON(strings.NewReader(stakingABIJSON))
	require.NoError(t, err)

	stakingAddress := common.HexToAddress("0x0")
	stakingConfig := &StakingConfig{
		EthRpcUrl:      "http://localhost:8545",
		StakingAddress: stakingAddress.Hex(),
		BlocksPerEpoch: 1,
	}

	pubKey, _, err := ed25519.GenKeys()
	require.NoError(t, err)

	mockEthClient := &MockEthereumClient{
		callContractFn: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			return nil, fmt.Errorf("failed to call contract")
		},
	}

	chain := &Chain{
		broadcaster:    &MockBroadcaster{},
		batchPool:      &MockBatchPool{},
		signerID:       pubKey,
		stakingConfig:  stakingConfig,
		height:         1,
		log:            slog.With("module", "dagger"),
		ethClient:      mockEthClient,
		stakingABI:     stakingABI,
		stakingAddress: stakingAddress,
	}

	err = chain.updateIncluders()
	require.Error(t, err, "failed to call contract")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = chain.startRound(ctx)
	require.Error(t, err, "failed to update includers: failed to call contract")
}

func TestChain_UpdateIncluders_Success(t *testing.T) {
	stakingABI, err := abi.JSON(strings.NewReader(stakingABIJSON))
	require.NoError(t, err)

	stakingAddress := common.HexToAddress("0x0")
	stakingConfig := &StakingConfig{
		EthRpcUrl:      "http://localhost:8545",
		StakingAddress: stakingAddress.Hex(),
		BlocksPerEpoch: 1,
	}

	pubKey, _, err := ed25519.GenKeys()
	require.NoError(t, err)

	addresses := []common.Address{common.HexToAddress("0x1")}
	pubKeys := []common.Hash{common.BytesToHash([]byte{0x01})}
	result, err := stakingABI.Pack("getStakers", addresses, pubKeys)
	require.NoError(t, err)

	mockEthClient := &MockEthereumClient{
		callContractFn: func(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
			return result, nil
		},
	}

	chain := &Chain{
		broadcaster:    &MockBroadcaster{},
		batchPool:      &MockBatchPool{},
		signerID:       pubKey,
		stakingConfig:  stakingConfig,
		height:         1,
		log:            slog.With("module", "dagger"),
		ethClient:      mockEthClient,
		stakingABI:     stakingABI,
		stakingAddress: stakingAddress,
	}

	err = chain.updateIncluders()
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = chain.startRound(ctx)
	require.NoError(t, err)
}

func TestChain_StartRound(t *testing.T) {
	const (
		nodeCount = 10
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	t.Cleanup(cancel)

	net, err := mocknet.FullMeshLinked(nodeCount)
	require.NoError(t, err)

	signers := make([]crypto.Signer, nodeCount)
	for i := 0; i < nodeCount; i++ {
		_, privKey, err := ed25519.GenKeys()
		require.NoError(t, err)
		signers[i] = &testSigner{privKey: privKey}
	}

	broadcasters := make([]*MockBroadcaster, nodeCount)
	pools := make([]*bapl.MulticastPool, nodeCount)
	for i, host := range net.Hosts() {
		broadcasters[i] = NewMockBroadcaster(t, host)
		pools[i] = NewMulticastPool(t, host, signers[i])
	}

	stakingConfig := &StakingConfig{
		EthRpcUrl:      "http://localhost:8545",
		StakingAddress: "0x0",
		BlocksPerEpoch: 1,
	}

	for i, broadcaster := range broadcasters {
		chain := NewChainWithStakingConfig(broadcaster, pools[i], stakingConfig, signers[i].ID())
		require.NotNil(t, chain)
		chain.Start()
		defer chain.Stop()

		err = chain.startRound(ctx)
		require.NoError(t, err)
	}
}

func NewMockBroadcaster(t *testing.T, host host.Host) *MockBroadcaster {
	psub, err := pubsub.NewGossipSub(context.Background(), host, pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign))
	require.NoError(t, err)
	return &MockBroadcaster{}
}

func NewMulticastPool(t *testing.T, host host.Host, signer crypto.Signer) *bapl.MulticastPool {
	mem := bapl.NewMemPool()
	mcast := bapl.NewMulticastPool(mem, host, mocknet.Peers, signer, &verifier{})
	mcast.Start()
	t.Cleanup(func() { mcast.Stop() })
	return mcast
}

type verifier struct{}

func (v verifier) Verify(ctx context.Context, batch *bapl.Batch) (bool, error) {
	return true, nil
}

// Test utilities

type testSigner struct {
	privKey ed25519.PrivateKey
}

func (t *testSigner) ID() []byte {
	return t.privKey.Public().(ed25519.PublicKey)
}

func newTestSigner() *testSigner {
	_, privKey, err := ed25519.GenKeys()
	if err != nil {
		panic(err)
	}

	return &testSigner{
		privKey: privKey,
	}
}

func (t *testSigner) Sign(data []byte) (crypto.Signature, error) {
	sig, err := t.privKey.Sign(rand.Reader, data, crypto.Hash(0))
	if err != nil {
		return crypto.Signature{}, err
	}

	return crypto.Signature{
		Body:   sig,
		Signer: t.privKey.Public().(ed25519.PublicKey),
	}, nil
}

func (t *testSigner) Verify(data []byte, sig crypto.Signature) error {
	pubKey := ed25519.PublicKey(sig.Signer)
	if !ed25519.Verify(pubKey, data, sig.Body) {
		return fmt.Errorf("invalid signature")
	}
	return nil
}

type testHasher struct{}

func (t *testHasher) Hash(msg rebro.Message) ([]byte, error) {
	h := sha256.New()
	_, err := h.Write(msg.Data)
	return h.Sum(nil), err
}

type testCertifier struct{}

func (t testCertifier) Certify(ctx context.Context, msg rebro.Message) error {
	return nil
}
