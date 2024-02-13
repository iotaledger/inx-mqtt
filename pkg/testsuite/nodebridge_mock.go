//nolint:revive,nilnil,structcheck,containedctx // skip linter for this package name
package testsuite

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/iotaledger/hive.go/runtime/event"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
	"github.com/iotaledger/iota.go/v4/nodeclient"
)

type MockedNodeBridge struct {
	t *testing.T

	events      *nodebridge.Events
	apiProvider iotago.APIProvider

	mockedLatestCommitment          *nodebridge.Commitment
	mockedLatestFinalizedCommitment *nodebridge.Commitment

	mockedBlocks              map[iotago.BlockID]*iotago.Block
	mockedBlockMetadata       map[iotago.BlockID]*api.BlockMetadataResponse
	mockedTransactionMetadata map[iotago.TransactionID]*api.TransactionMetadataResponse
	mockedOutputs             map[iotago.OutputID]*nodebridge.Output

	mockedStreamListenToBlocks               *MockedStream[MockedBlock]
	mockedStreamListenToAcceptedBlocks       *MockedStream[inx.BlockMetadata]
	mockedStreamListenToConfirmedBlocks      *MockedStream[inx.BlockMetadata]
	mockedStreamListenToCommitments          *MockedStream[MockedCommitment]
	mockedStreamListenToLedgerUpdates        *MockedStream[nodebridge.LedgerUpdate]
	mockedStreamListenToAcceptedTransactions *MockedStream[nodebridge.AcceptedTransaction]
}

var _ nodebridge.NodeBridge = &MockedNodeBridge{}

func NewMockedNodeBridge(t *testing.T, iotaAPI iotago.API) *MockedNodeBridge {
	t.Helper()

	return &MockedNodeBridge{
		t: t,
		events: &nodebridge.Events{
			LatestCommitmentChanged:          event.New1[*nodebridge.Commitment](),
			LatestFinalizedCommitmentChanged: event.New1[*nodebridge.Commitment](),
		},
		apiProvider:               iotago.SingleVersionProvider(iotaAPI),
		mockedBlocks:              make(map[iotago.BlockID]*iotago.Block),
		mockedBlockMetadata:       make(map[iotago.BlockID]*api.BlockMetadataResponse),
		mockedTransactionMetadata: make(map[iotago.TransactionID]*api.TransactionMetadataResponse),
		mockedOutputs:             make(map[iotago.OutputID]*nodebridge.Output),
	}
}

//
// NodeBridge interface
//

func (m *MockedNodeBridge) Events() *nodebridge.Events {
	return m.events
}

func (m *MockedNodeBridge) Connect(ctx context.Context, address string, maxConnectionAttempts uint) error {
	panic("not implemented")
}

func (m *MockedNodeBridge) Run(ctx context.Context) {
	panic("not implemented")
}

func (m *MockedNodeBridge) Client() inx.INXClient {
	panic("not implemented")
}

func (m *MockedNodeBridge) NodeConfig() *inx.NodeConfiguration {
	panic("not implemented")
}

func (m *MockedNodeBridge) APIProvider() iotago.APIProvider {
	return m.apiProvider
}

func (m *MockedNodeBridge) INXNodeClient() (*nodeclient.Client, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) Management(ctx context.Context) (nodeclient.ManagementClient, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) Indexer(ctx context.Context) (nodeclient.IndexerClient, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) EventAPI(ctx context.Context) (*nodeclient.EventAPIClient, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) BlockIssuer(ctx context.Context) (nodeclient.BlockIssuerClient, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) ReadIsCandidate(ctx context.Context, id iotago.AccountID, slot iotago.SlotIndex) (bool, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) ReadIsCommitteeMember(ctx context.Context, id iotago.AccountID, slot iotago.SlotIndex) (bool, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) ReadIsValidatorAccount(ctx context.Context, id iotago.AccountID, slot iotago.SlotIndex) (bool, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) RegisterAPIRoute(ctx context.Context, route string, bindAddress string, path string) error {
	return nil
}

func (m *MockedNodeBridge) UnregisterAPIRoute(ctx context.Context, route string) error {
	return nil
}

func (m *MockedNodeBridge) ActiveRootBlocks(ctx context.Context) (map[iotago.BlockID]iotago.CommitmentID, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) SubmitBlock(ctx context.Context, block *iotago.Block) (iotago.BlockID, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) Block(ctx context.Context, blockID iotago.BlockID) (*iotago.Block, error) {
	if block, ok := m.mockedBlocks[blockID]; ok {
		return block, nil
	}

	return nil, status.Errorf(codes.NotFound, "block %s not found", blockID.ToHex())
}

func (m *MockedNodeBridge) BlockMetadata(ctx context.Context, blockID iotago.BlockID) (*api.BlockMetadataResponse, error) {
	if blockMetadata, ok := m.mockedBlockMetadata[blockID]; ok {
		return blockMetadata, nil
	}

	return nil, status.Errorf(codes.NotFound, "metadata for block %s not found", blockID.ToHex())
}

func (m *MockedNodeBridge) ListenToBlocks(ctx context.Context, consumer func(block *iotago.Block, rawData []byte) error) error {
	if m.mockedStreamListenToBlocks == nil {
		require.FailNow(m.t, "ListenToBlocks mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToBlocks.receiverFunc(), func(block *MockedBlock) error {
		return consumer(block.Block, block.RawBlockData)
	})
	require.NoError(m.t, err, "ListenToBlocks failed")

	return nil
}

func (m *MockedNodeBridge) ListenToAcceptedBlocks(ctx context.Context, consumer func(blockMetadata *api.BlockMetadataResponse) error) error {
	if m.mockedStreamListenToAcceptedBlocks == nil {
		require.FailNow(m.t, "ListenToAcceptedBlocks mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToAcceptedBlocks.receiverFunc(), func(inxBlockMetadata *inx.BlockMetadata) error {
		blockMetadata, err := inxBlockMetadata.Unwrap()
		if err != nil {
			return err
		}

		return consumer(blockMetadata)
	})
	require.NoError(m.t, err, "ListenToAcceptedBlocks failed")

	return nil
}

func (m *MockedNodeBridge) ListenToConfirmedBlocks(ctx context.Context, consumer func(blockMetadata *api.BlockMetadataResponse) error) error {
	if m.mockedStreamListenToConfirmedBlocks == nil {
		require.FailNow(m.t, "ListenToConfirmedBlocks mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToConfirmedBlocks.receiverFunc(), func(inxBlockMetadata *inx.BlockMetadata) error {
		blockMetadata, err := inxBlockMetadata.Unwrap()
		if err != nil {
			return err
		}

		return consumer(blockMetadata)
	})
	require.NoError(m.t, err, "ListenToConfirmedBlocks failed")

	return nil
}

// TransactionMetadata returns the transaction metadata for the given transaction ID.
func (m *MockedNodeBridge) TransactionMetadata(ctx context.Context, transactionID iotago.TransactionID) (*api.TransactionMetadataResponse, error) {
	if transactionMetadata, ok := m.mockedTransactionMetadata[transactionID]; ok {
		return transactionMetadata, nil
	}

	return nil, status.Errorf(codes.NotFound, "metadata for transaction %s not found", transactionID.ToHex())
}

func (m *MockedNodeBridge) Output(ctx context.Context, outputID iotago.OutputID) (*nodebridge.Output, error) {
	if output, ok := m.mockedOutputs[outputID]; ok {
		return output, nil
	}

	return nil, status.Errorf(codes.NotFound, "output %s not found", outputID.ToHex())
}

func (m *MockedNodeBridge) ForceCommitUntil(ctx context.Context, slot iotago.SlotIndex) error {
	panic("not implemented")
}

func (m *MockedNodeBridge) Commitment(ctx context.Context, slot iotago.SlotIndex) (*nodebridge.Commitment, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) CommitmentByID(ctx context.Context, id iotago.CommitmentID) (*nodebridge.Commitment, error) {
	panic("not implemented")
}

func (m *MockedNodeBridge) ListenToCommitments(ctx context.Context, startSlot, endSlot iotago.SlotIndex, consumer func(commitment *nodebridge.Commitment, rawData []byte) error) error {
	if m.mockedStreamListenToCommitments == nil {
		require.FailNow(m.t, "ListenToCommitments mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToCommitments.receiverFunc(), func(commitment *MockedCommitment) error {
		return consumer(&nodebridge.Commitment{
			CommitmentID: commitment.CommitmentID,
			Commitment:   commitment.Commitment,
		}, commitment.RawCommitmentData)
	})
	require.NoError(m.t, err, "ListenToCommitments failed")

	return nil
}

func (m *MockedNodeBridge) ListenToLedgerUpdates(ctx context.Context, startSlot, endSlot iotago.SlotIndex, consumer func(update *nodebridge.LedgerUpdate) error) error {
	if m.mockedStreamListenToLedgerUpdates == nil {
		require.FailNow(m.t, "ListenToLedgerUpdates mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToLedgerUpdates.receiverFunc(), consumer)
	require.NoError(m.t, err, "ListenToLedgerUpdates failed")

	return nil
}

func (m *MockedNodeBridge) ListenToAcceptedTransactions(ctx context.Context, consumer func(tx *nodebridge.AcceptedTransaction) error) error {
	if m.mockedStreamListenToAcceptedTransactions == nil {
		require.FailNow(m.t, "ListenToAcceptedTransactions mock not initialized")
	}

	err := nodebridge.ListenToStream(ctx, m.mockedStreamListenToAcceptedTransactions.receiverFunc(), consumer)
	require.NoError(m.t, err, "ListenToAcceptedTransactions failed")

	return nil
}

func (m *MockedNodeBridge) NodeStatus() *inx.NodeStatus {
	panic("not implemented")
}

func (m *MockedNodeBridge) IsNodeHealthy() bool {
	panic("not implemented")
}

func (m *MockedNodeBridge) LatestCommitment() *nodebridge.Commitment {
	return m.mockedLatestCommitment
}

func (m *MockedNodeBridge) LatestFinalizedCommitment() *nodebridge.Commitment {
	return m.mockedLatestFinalizedCommitment
}

func (m *MockedNodeBridge) PruningEpoch() iotago.EpochIndex {
	panic("not implemented")
}

func (m *MockedNodeBridge) RequestTips(ctx context.Context, count uint32) (strong iotago.BlockIDs, weak iotago.BlockIDs, shallowLike iotago.BlockIDs, err error) {
	panic("not implemented")
}

//
// Mock functions
//

func (m *MockedNodeBridge) MockClear() {
	m.mockedBlocks = make(map[iotago.BlockID]*iotago.Block)
	m.mockedBlockMetadata = make(map[iotago.BlockID]*api.BlockMetadataResponse)
	m.mockedOutputs = make(map[iotago.OutputID]*nodebridge.Output)

	if m.mockedStreamListenToBlocks != nil {
		m.mockedStreamListenToBlocks.Close()
		m.mockedStreamListenToBlocks = nil
	}
	if m.mockedStreamListenToAcceptedBlocks != nil {
		m.mockedStreamListenToAcceptedBlocks.Close()
		m.mockedStreamListenToAcceptedBlocks = nil
	}
	if m.mockedStreamListenToConfirmedBlocks != nil {
		m.mockedStreamListenToConfirmedBlocks.Close()
		m.mockedStreamListenToConfirmedBlocks = nil
	}
	if m.mockedStreamListenToCommitments != nil {
		m.mockedStreamListenToCommitments.Close()
		m.mockedStreamListenToCommitments = nil
	}
	if m.mockedStreamListenToLedgerUpdates != nil {
		m.mockedStreamListenToLedgerUpdates.Close()
		m.mockedStreamListenToLedgerUpdates = nil
	}
	if m.mockedStreamListenToAcceptedTransactions != nil {
		m.mockedStreamListenToAcceptedTransactions.Close()
		m.mockedStreamListenToAcceptedTransactions = nil
	}
}

func (m *MockedNodeBridge) MockSetLatestCommitment(commitment *nodebridge.Commitment) {
	m.mockedLatestCommitment = commitment
}

func (m *MockedNodeBridge) MockSetLatestFinalizedCommitment(commitment *nodebridge.Commitment) {
	m.mockedLatestFinalizedCommitment = commitment
}

func (m *MockedNodeBridge) MockReceiveLatestCommitment(commitment *nodebridge.Commitment) {
	m.mockedLatestCommitment = commitment
	m.Events().LatestCommitmentChanged.Trigger(commitment)
}

func (m *MockedNodeBridge) MockReceiveLatestFinalizedCommitment(commitment *nodebridge.Commitment) {
	m.mockedLatestFinalizedCommitment = commitment
	m.Events().LatestFinalizedCommitmentChanged.Trigger(commitment)
}

func (m *MockedNodeBridge) MockAddBlock(blockID iotago.BlockID, block *iotago.Block) {
	m.mockedBlocks[blockID] = block
}

func (m *MockedNodeBridge) MockAddBlockMetadata(blockID iotago.BlockID, blockMetadata *api.BlockMetadataResponse) {
	m.mockedBlockMetadata[blockID] = blockMetadata
}

func (m *MockedNodeBridge) MockAddTransactionMetadata(transactionID iotago.TransactionID, transactionMetadata *api.TransactionMetadataResponse) {
	m.mockedTransactionMetadata[transactionID] = transactionMetadata
}

func (m *MockedNodeBridge) MockAddOutput(outputID iotago.OutputID, output *nodebridge.Output) {
	m.mockedOutputs[outputID] = output
}

type MockedBlock struct {
	Block        *iotago.Block
	RawBlockData []byte
}

type MockedCommitment struct {
	CommitmentID      iotago.CommitmentID
	Commitment        *iotago.Commitment
	RawCommitmentData []byte
}

func (m *MockedNodeBridge) MockListenToBlocks() *MockedStream[MockedBlock] {
	m.mockedStreamListenToBlocks = InitMockedStream[MockedBlock]()
	return m.mockedStreamListenToBlocks
}

func (m *MockedNodeBridge) MockListenToAcceptedBlocks() *MockedStream[inx.BlockMetadata] {
	m.mockedStreamListenToAcceptedBlocks = InitMockedStream[inx.BlockMetadata]()
	return m.mockedStreamListenToAcceptedBlocks
}

func (m *MockedNodeBridge) MockListenToConfirmedBlocks() *MockedStream[inx.BlockMetadata] {
	m.mockedStreamListenToConfirmedBlocks = InitMockedStream[inx.BlockMetadata]()
	return m.mockedStreamListenToConfirmedBlocks
}

func (m *MockedNodeBridge) MockListenToCommitments() *MockedStream[MockedCommitment] {
	m.mockedStreamListenToCommitments = InitMockedStream[MockedCommitment]()
	return m.mockedStreamListenToCommitments
}

func (m *MockedNodeBridge) MockListenToLedgerUpdates() *MockedStream[nodebridge.LedgerUpdate] {
	m.mockedStreamListenToLedgerUpdates = InitMockedStream[nodebridge.LedgerUpdate]()
	return m.mockedStreamListenToLedgerUpdates
}

func (m *MockedNodeBridge) MockListenToAcceptedTransactions() *MockedStream[nodebridge.AcceptedTransaction] {
	m.mockedStreamListenToAcceptedTransactions = InitMockedStream[nodebridge.AcceptedTransaction]()
	return m.mockedStreamListenToAcceptedTransactions
}

type MockedStream[T any] struct {
	ctx          context.Context
	cancel       context.CancelFunc
	receiverChan chan *T
}

func InitMockedStream[T any]() *MockedStream[T] {
	ctx, cancel := context.WithCancel(context.Background())
	receiverChan := make(chan *T)

	return &MockedStream[T]{
		ctx:          ctx,
		cancel:       cancel,
		receiverChan: receiverChan,
	}
}

func (m *MockedStream[T]) receiverFunc() func() (*T, error) {
	return func() (*T, error) {
		select {
		case <-m.ctx.Done():
			return nil, io.EOF

		case obj, ok := <-m.receiverChan:
			if !ok {
				return nil, io.EOF
			}

			return obj, nil
		}
	}
}

func (m *MockedStream[T]) Receive(obj *T) {
	m.receiverChan <- obj
}

func (m *MockedStream[T]) Close() {
	m.cancel()
	close(m.receiverChan)
}
