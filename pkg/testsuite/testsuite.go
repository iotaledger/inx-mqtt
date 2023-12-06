//nolint:contextcheck,forcetypeassert,exhaustive
package testsuite

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/hive.go/logger"
	"github.com/iotaledger/hive.go/runtime/options"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
	"github.com/iotaledger/iota.go/v4/tpkg"
)

type TestSuite struct {
	T *testing.T

	api        iotago.API
	nodeBridge *MockedNodeBridge
	broker     *MockedBroker
	server     *mqtt.Server

	mockedStreamListenToBlocks               *MockedStream[MockedBlock]
	mockedStreamListenToAcceptedBlocks       *MockedStream[inx.BlockMetadata]
	mockedStreamListenToConfirmedBlocks      *MockedStream[inx.BlockMetadata]
	mockedStreamListenToCommitments          *MockedStream[MockedCommitment]
	mockedStreamListenToLedgerUpdates        *MockedStream[nodebridge.LedgerUpdate]
	mockedStreamListenToAcceptedTransactions *MockedStream[nodebridge.AcceptedTransaction]
}

func NewTestSuite(t *testing.T) *TestSuite {
	t.Helper()

	rootLogger, err := logger.NewRootLogger(logger.DefaultCfg)
	require.NoError(t, err)

	api := tpkg.ZeroCostTestAPI

	bridge := NewMockedNodeBridge(t, api)
	broker := NewMockedBroker(t)
	server, err := mqtt.NewServer(
		rootLogger.Named(t.Name()),
		bridge,
		broker,
		nil,
	)
	require.NoError(t, err)

	return &TestSuite{
		T:          t,
		api:        api,
		nodeBridge: bridge,
		broker:     broker,
		server:     server,

		mockedStreamListenToBlocks:               bridge.MockListenToBlocks(),
		mockedStreamListenToAcceptedBlocks:       bridge.MockListenToAcceptedBlocks(),
		mockedStreamListenToConfirmedBlocks:      bridge.MockListenToConfirmedBlocks(),
		mockedStreamListenToCommitments:          bridge.MockListenToCommitments(),
		mockedStreamListenToLedgerUpdates:        bridge.MockListenToLedgerUpdates(),
		mockedStreamListenToAcceptedTransactions: bridge.MockListenToAcceptedTransactions(),
	}
}

func (ts *TestSuite) Run(ctx context.Context) {
	err := ts.server.Start(ctx)
	require.NoError(ts.T, err)

	go func() {
		<-ctx.Done()

		err = ts.server.Stop()
		require.NoError(ts.T, err)
	}()
}

func (ts *TestSuite) API() iotago.API {
	return ts.api
}

func (ts *TestSuite) Reset() {
	ts.nodeBridge.MockClear()
	ts.broker.MockClear()

	ts.mockedStreamListenToBlocks = ts.nodeBridge.MockListenToBlocks()
	ts.mockedStreamListenToAcceptedBlocks = ts.nodeBridge.MockListenToAcceptedBlocks()
	ts.mockedStreamListenToConfirmedBlocks = ts.nodeBridge.MockListenToConfirmedBlocks()
	ts.mockedStreamListenToCommitments = ts.nodeBridge.MockListenToCommitments()
	ts.mockedStreamListenToLedgerUpdates = ts.nodeBridge.MockListenToLedgerUpdates()
	ts.mockedStreamListenToAcceptedTransactions = ts.nodeBridge.MockListenToAcceptedTransactions()
}

//
// NodeBridge
//

func (ts *TestSuite) SetLatestCommitment(commitment *nodebridge.Commitment) {
	ts.nodeBridge.MockSetLatestCommitment(commitment)
}

func (ts *TestSuite) SetLatestFinalizedCommitment(commitment *nodebridge.Commitment) {
	ts.nodeBridge.MockSetLatestFinalizedCommitment(commitment)
}

func (ts *TestSuite) ReceiveLatestCommitment(commitment *nodebridge.Commitment) {
	ts.nodeBridge.MockReceiveLatestCommitment(commitment)
}

func (ts *TestSuite) ReceiveLatestFinalizedCommitment(commitment *nodebridge.Commitment) {
	ts.nodeBridge.MockReceiveLatestFinalizedCommitment(commitment)
}

func (ts *TestSuite) MockAddBlock(blockID iotago.BlockID, block *iotago.Block) {
	ts.nodeBridge.MockAddBlock(blockID, block)
}

func (ts *TestSuite) MockAddBlockMetadata(blockID iotago.BlockID, blockMetadata *api.BlockMetadataResponse) {
	ts.nodeBridge.MockAddBlockMetadata(blockID, blockMetadata)
}

func (ts *TestSuite) MockAddOutput(outputID iotago.OutputID, output *nodebridge.Output) {
	ts.nodeBridge.MockAddOutput(outputID, output)
}

func (ts *TestSuite) ReceiveBlock(block *MockedBlock) {
	ts.mockedStreamListenToBlocks.Receive(block)
}

func (ts *TestSuite) ReceiveAcceptedBlock(metadata *inx.BlockMetadata) {
	ts.mockedStreamListenToAcceptedBlocks.Receive(metadata)
}

func (ts *TestSuite) ReceiveConfirmedBlock(metadata *inx.BlockMetadata) {
	ts.mockedStreamListenToConfirmedBlocks.Receive(metadata)
}

func (ts *TestSuite) ReceiveCommitment(commitment *MockedCommitment) {
	ts.mockedStreamListenToCommitments.Receive(commitment)
}

func (ts *TestSuite) ReceiveLedgerUpdate(update *nodebridge.LedgerUpdate) {
	ts.mockedStreamListenToLedgerUpdates.Receive(update)
}

func (ts *TestSuite) ReceiveAcceptedTransaction(tx *nodebridge.AcceptedTransaction) {
	ts.mockedStreamListenToAcceptedTransactions.Receive(tx)
}

//
// MQTT Broker
//

func (ts *TestSuite) MQTTSetHasSubscribersCallback(callback func(topic string)) {
	ts.broker.MockSetHasSubscribersCallback(callback)
}

func (ts *TestSuite) MQTTClientConnect(clientID string) {
	ts.broker.MockClientConnected(clientID)
}

func (ts *TestSuite) MQTTClientDisconnect(clientID string) {
	ts.broker.MockClientDisconnected(clientID)
}

func (ts *TestSuite) MQTTSubscribe(clientID string, topic string, callback func(topic string, payload []byte)) (unsubscribe func()) {
	ts.broker.MockTopicSubscribed(clientID, topic, callback)

	return func() {
		ts.broker.MockTopicUnsubscribed(clientID, topic)
	}
}

func (ts *TestSuite) MQTTUnsubscribe(clientID string, topic string) {
	ts.broker.MockTopicUnsubscribed(clientID, topic)
}

//
// Utility functions
//

type TestTransaction struct {
	ConsumedOutputCreationTransaction *iotago.Transaction
	ConsumedOutputID                  iotago.OutputID
	Transaction                       *iotago.Transaction
	TransactionID                     iotago.TransactionID
	Output                            *nodebridge.Output
	OutputID                          iotago.OutputID
	OutputWithMetadataResponse        *api.OutputWithMetadataResponse
	SenderAddress                     iotago.Address
	OwnerAddress                      iotago.Address
	Block                             *iotago.Block
	BlockID                           iotago.BlockID
}

func (ts *TestSuite) NewTestTransaction(fromSameAddress bool, opts ...options.Option[iotago.Transaction]) *TestTransaction {
	// we need to create the transaction first to apply the options, so we can simplify the test by sending from the same address
	transaction := &iotago.Transaction{
		API: ts.API(),
		TransactionEssence: &iotago.TransactionEssence{
			NetworkID:     ts.API().ProtocolParameters().NetworkID(),
			CreationSlot:  tpkg.RandSlot(),
			ContextInputs: nil,
			// we set those later
			Inputs:       iotago.TxEssenceInputs{},
			Allotments:   nil,
			Capabilities: nil,
			Payload:      nil,
		},
		Outputs: iotago.Outputs[iotago.TxEssenceOutput]{
			tpkg.RandOutput(iotago.OutputBasic).(*iotago.BasicOutput),
		},
	}
	options.Apply(transaction, opts)

	createdOutput := transaction.Outputs[0]

	var ownerAddress iotago.Address
	switch createdOutput.Type() {
	case iotago.OutputAnchor:
		ownerAddress = createdOutput.UnlockConditionSet().StateControllerAddress().Address
	case iotago.OutputFoundry:
		ownerAddress = createdOutput.UnlockConditionSet().ImmutableAccount().Address
	default:
		ownerAddress = createdOutput.UnlockConditionSet().Address().Address
	}

	// simplify the test by sending from the same address (less topics to ignore)
	consumedOutput := tpkg.RandOutput(iotago.OutputBasic).(*iotago.BasicOutput)
	if fromSameAddress {
		consumedOutput.UnlockConditionSet().Address().Address = ownerAddress
	}
	senderAddress := consumedOutput.UnlockConditionSet().Address().Address

	consumedOutputCreationTransaction := &iotago.Transaction{
		API: ts.API(),
		TransactionEssence: &iotago.TransactionEssence{
			NetworkID:     ts.API().ProtocolParameters().NetworkID(),
			CreationSlot:  tpkg.RandSlot(),
			ContextInputs: nil,
			Inputs: iotago.TxEssenceInputs{
				tpkg.RandUTXOInput(),
			},
			Allotments:   nil,
			Capabilities: nil,
			Payload:      nil,
		},
		Outputs: iotago.Outputs[iotago.TxEssenceOutput]{
			consumedOutput,
		},
	}
	consumedOutputID := iotago.OutputIDFromTransactionIDAndIndex(lo.PanicOnErr(consumedOutputCreationTransaction.ID()), 0)

	// now we can set the correct inputs
	transaction.TransactionEssence.Inputs = iotago.TxEssenceInputs{
		consumedOutputID.UTXOInput(),
	}
	transactionID := lo.PanicOnErr(transaction.ID())

	block := tpkg.RandBlock(
		tpkg.RandBasicBlockBodyWithPayload(ts.API(),
			tpkg.RandSignedTransactionWithTransaction(ts.API(),
				transaction,
			),
		), ts.API(), iotago.Mana(500))
	blockID := block.MustID()

	output := ts.NewNodeBridgeOutputFromTransaction(blockID, transaction)

	return &TestTransaction{
		ConsumedOutputCreationTransaction: consumedOutputCreationTransaction,
		ConsumedOutputID:                  consumedOutputID,
		Transaction:                       transaction,
		TransactionID:                     transactionID,
		Output:                            output,
		OutputID:                          output.OutputID,
		OutputWithMetadataResponse:        ts.NewOutputWithMetadataResponseFromTransaction(blockID, transaction),
		SenderAddress:                     senderAddress,
		OwnerAddress:                      ownerAddress,
		Block:                             block,
		BlockID:                           blockID,
	}
}

func (ts *TestSuite) NewOutputWithMetadataResponseFromTransaction(blockID iotago.BlockID, transaction *iotago.Transaction) *api.OutputWithMetadataResponse {
	transactionID := lo.PanicOnErr(transaction.ID())
	output := transaction.Outputs[0]
	outputID := iotago.OutputIDFromTransactionIDAndIndex(transactionID, 0)
	outputIDProof := lo.PanicOnErr(iotago.OutputIDProofFromTransaction(transaction, 0))

	return &api.OutputWithMetadataResponse{
		Output:        output,
		OutputIDProof: outputIDProof,
		Metadata: &api.OutputMetadata{
			OutputID: outputID,
			BlockID:  blockID,
			Included: &api.OutputInclusionMetadata{
				Slot:          blockID.Slot(),
				TransactionID: transactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			Spent:              nil,
			LatestCommitmentID: tpkg.RandCommitmentID(),
		},
	}
}

func (ts *TestSuite) NewSpentOutputWithMetadataResponseFromTransaction(creationBlockID iotago.BlockID, creationTx *iotago.Transaction, spendBlockID iotago.BlockID, spendTx *iotago.Transaction) *api.OutputWithMetadataResponse {
	creationTransactionID := lo.PanicOnErr(creationTx.ID())
	spendTransactionID := lo.PanicOnErr(spendTx.ID())

	output := creationTx.Outputs[0]
	outputID := iotago.OutputIDFromTransactionIDAndIndex(creationTransactionID, 0)
	outputIDProof := lo.PanicOnErr(iotago.OutputIDProofFromTransaction(creationTx, 0))

	return &api.OutputWithMetadataResponse{
		Output:        output,
		OutputIDProof: outputIDProof,
		Metadata: &api.OutputMetadata{
			OutputID: outputID,
			BlockID:  creationBlockID,
			Included: &api.OutputInclusionMetadata{
				Slot:          creationBlockID.Slot(),
				TransactionID: creationTransactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			Spent: &api.OutputConsumptionMetadata{
				Slot:          spendBlockID.Slot(),
				TransactionID: spendTransactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			LatestCommitmentID: tpkg.RandCommitmentID(),
		},
	}
}

func (ts *TestSuite) NewNodeBridgeOutputFromTransaction(blockID iotago.BlockID, transaction *iotago.Transaction) *nodebridge.Output {
	transactionID := lo.PanicOnErr(transaction.ID())
	output := transaction.Outputs[0]
	outputID := iotago.OutputIDFromTransactionIDAndIndex(transactionID, 0)
	outputIDProof := lo.PanicOnErr(iotago.OutputIDProofFromTransaction(transaction, 0))

	return &nodebridge.Output{
		OutputID:      outputID,
		Output:        output,
		OutputIDProof: outputIDProof,
		Metadata: &api.OutputMetadata{
			OutputID: outputID,
			BlockID:  blockID,
			Included: &api.OutputInclusionMetadata{
				Slot:          blockID.Slot(),
				TransactionID: transactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			Spent:              nil,
			LatestCommitmentID: tpkg.RandCommitmentID(),
		},
		RawOutputData: lo.PanicOnErr(ts.API().Encode(output)),
	}
}

func (ts *TestSuite) NewNodeBridgeOutputFromOutputWithMetadata(outputWithMetadata *api.OutputWithMetadataResponse) *nodebridge.Output {
	return &nodebridge.Output{
		OutputID:      outputWithMetadata.Metadata.OutputID,
		Output:        outputWithMetadata.Output,
		OutputIDProof: outputWithMetadata.OutputIDProof,
		Metadata:      outputWithMetadata.Metadata,
		RawOutputData: lo.PanicOnErr(ts.API().Encode(outputWithMetadata.Output)),
	}
}

func (ts *TestSuite) NewSpentNodeBridgeOutputFromTransaction(creationBlockID iotago.BlockID, creationTx *iotago.Transaction, spendSlot iotago.SlotIndex, spendTransactionID iotago.TransactionID) *nodebridge.Output {
	creationTransactionID := lo.PanicOnErr(creationTx.ID())

	output := creationTx.Outputs[0]
	outputID := iotago.OutputIDFromTransactionIDAndIndex(creationTransactionID, 0)
	outputIDProof := lo.PanicOnErr(iotago.OutputIDProofFromTransaction(creationTx, 0))

	return &nodebridge.Output{
		OutputID:      outputID,
		Output:        output,
		OutputIDProof: outputIDProof,
		Metadata: &api.OutputMetadata{
			OutputID: outputID,
			BlockID:  creationBlockID,
			Included: &api.OutputInclusionMetadata{
				Slot:          creationBlockID.Slot(),
				TransactionID: creationTransactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			Spent: &api.OutputConsumptionMetadata{
				Slot:          spendSlot,
				TransactionID: spendTransactionID,
				CommitmentID:  tpkg.RandCommitmentID(),
			},
			LatestCommitmentID: tpkg.RandCommitmentID(),
		},
		RawOutputData: lo.PanicOnErr(ts.API().Encode(output)),
	}
}

func (ts *TestSuite) NewSpentNodeBridgeOutputFromOutputWithMetadata(outputWithMetadata *api.OutputWithMetadataResponse, spendSlot iotago.SlotIndex, spendTransactionID iotago.TransactionID) *nodebridge.Output {
	outputWithMetadata.Metadata.Spent = &api.OutputConsumptionMetadata{
		Slot:          spendSlot,
		TransactionID: spendTransactionID,
		CommitmentID:  tpkg.RandCommitmentID(),
	}

	return &nodebridge.Output{
		OutputID:      outputWithMetadata.Metadata.OutputID,
		Output:        outputWithMetadata.Output,
		OutputIDProof: outputWithMetadata.OutputIDProof,
		Metadata:      outputWithMetadata.Metadata,
		RawOutputData: lo.PanicOnErr(ts.API().Encode(outputWithMetadata.Output)),
	}
}
