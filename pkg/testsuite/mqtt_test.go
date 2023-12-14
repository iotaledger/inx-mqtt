//nolint:forcetypeassert,scopelint,goconst,dupl
package testsuite_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/iotaledger/hive.go/lo"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	"github.com/iotaledger/inx-mqtt/pkg/mqtt"
	"github.com/iotaledger/inx-mqtt/pkg/testsuite"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v4"
	"github.com/iotaledger/iota.go/v4/api"
	"github.com/iotaledger/iota.go/v4/tpkg"
)

func TestMqttTopics(t *testing.T) {
	ts := testsuite.NewTestSuite(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ts.Run(ctx)

	type testTopic struct {
		// the topic to subscribe to
		topic string
		// indicates that the message is received during the subscription to the topic.
		// this is used to test the "polling mode".
		isPollingTarget bool
		// indicates that the message is received after the subscription to the topic.
		// this is used to test the "event driven mode".
		isEventTarget bool
	}

	type test struct {
		// the name of the test
		name string
		// the topics the test should subscribe to ("/raw" topics will be checked automatically)
		topics []*testTopic
		// the topics that should be ignored by the test (it's legit to receive messages on these topics)
		topicsIgnore []string
		// the expected JSON result received by the client on the subscribed topic
		jsonTarget []byte
		// the expected raw result received by the client on the subscribed topic
		rawTarget []byte
		// the function is called by the test before the MQTT topic is subscribed to (e.g. to inject test data)
		preSubscribeFunc func()
		// the function is called by the test after the MQTT topic is subscribed to (e.g. to inject test data)
		postSubscribeFunc func()
	}

	tests := []*test{

		// ok - LatestCommitment
		func() *test {
			commitment := tpkg.RandCommitment()

			return &test{
				name: "ok - LatestCommitment",
				topics: []*testTopic{
					{
						topic: mqtt.TopicCommitmentsLatest,
						// we receive the topic once during the subscription
						// and a second time when the commitment is received
						isPollingTarget: true,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{},
				jsonTarget:   lo.PanicOnErr(ts.API().JSONEncode(commitment)),
				rawTarget:    lo.PanicOnErr(ts.API().Encode(commitment)),
				preSubscribeFunc: func() {
					ts.SetLatestCommitment(&nodebridge.Commitment{
						CommitmentID: commitment.MustID(),
						Commitment:   commitment,
					})
				},
				postSubscribeFunc: func() {
					ts.ReceiveLatestCommitment(&nodebridge.Commitment{
						CommitmentID: commitment.MustID(),
						Commitment:   commitment,
					})
				},
			}
		}(),

		// ok - LatestFinalizedCommitment
		func() *test {
			commitment := tpkg.RandCommitment()

			return &test{
				name: "ok - LatestFinalizedCommitment",
				topics: []*testTopic{
					{
						topic: mqtt.TopicCommitmentsFinalized,
						// we receive the topic once during the subscription
						// and a second time when the commitment is received
						isPollingTarget: true,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{},
				jsonTarget:   lo.PanicOnErr(ts.API().JSONEncode(commitment)),
				rawTarget:    lo.PanicOnErr(ts.API().Encode(commitment)),
				preSubscribeFunc: func() {
					ts.SetLatestFinalizedCommitment(&nodebridge.Commitment{
						CommitmentID: commitment.MustID(),
						Commitment:   commitment,
					})
				},
				postSubscribeFunc: func() {
					ts.ReceiveLatestFinalizedCommitment(&nodebridge.Commitment{
						CommitmentID: commitment.MustID(),
						Commitment:   commitment,
					})
				},
			}
		}(),

		// ok - Validation block
		func() *test {
			block := tpkg.RandBlock(tpkg.RandValidationBlockBody(ts.API()), ts.API(), iotago.Mana(500))

			return &test{
				name: "ok - Validation block",
				topics: []*testTopic{
					{
						topic:           mqtt.TopicBlocks,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksValidation,
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{},
				jsonTarget:   lo.PanicOnErr(ts.API().JSONEncode(block)),
				rawTarget:    lo.PanicOnErr(ts.API().Encode(block)),
				postSubscribeFunc: func() {
					ts.ReceiveBlock(&testsuite.MockedBlock{
						Block:        block,
						RawBlockData: lo.PanicOnErr(ts.API().Encode(block)),
					})
				},
			}
		}(),

		// ok - Basic block with tagged data
		func() *test {
			block := tpkg.RandBlock(
				tpkg.RandBasicBlockBodyWithPayload(ts.API(),
					&iotago.TaggedData{
						Tag:  []byte("my tagged data payload"),
						Data: []byte("some nice data"),
					},
				), ts.API(), iotago.Mana(500))

			return &test{
				name: "ok - Basic block with tagged data",
				topics: []*testTopic{
					{
						topic:           mqtt.TopicBlocks,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksBasic,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksBasicTaggedData,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicBlocksBasicTaggedDataTag([]byte("my tagged data payload")),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{},
				jsonTarget:   lo.PanicOnErr(ts.API().JSONEncode(block)),
				rawTarget:    lo.PanicOnErr(ts.API().Encode(block)),
				postSubscribeFunc: func() {
					ts.ReceiveBlock(&testsuite.MockedBlock{
						Block:        block,
						RawBlockData: lo.PanicOnErr(ts.API().Encode(block)),
					})
				},
			}
		}(),

		// ok - Basic block with transaction and tagged data payload
		func() *test {
			testTx := ts.NewTestTransaction(true, tpkg.WithTxEssencePayload(
				&iotago.TaggedData{
					Tag:  []byte("my tagged data payload"),
					Data: []byte("some nice data"),
				},
			))

			return &test{
				name: "ok - Basic block with transaction and tagged data payload",
				topics: []*testTopic{
					{
						topic:           mqtt.TopicBlocks,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksBasic,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksBasicTransaction,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.TopicBlocksBasicTransactionTaggedData,
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicBlocksBasicTransactionTaggedDataTag([]byte("my tagged data payload")),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{},
				jsonTarget:   lo.PanicOnErr(ts.API().JSONEncode(testTx.Block)),
				rawTarget:    lo.PanicOnErr(ts.API().Encode(testTx.Block)),
				postSubscribeFunc: func() {
					ts.ReceiveBlock(&testsuite.MockedBlock{
						Block:        testTx.Block,
						RawBlockData: lo.PanicOnErr(ts.API().Encode(testTx.Block)),
					})
				},
			}
		}(),

		// ok - Basic block with transaction and tagged data payload - TransactionsIncludedBlockTopic
		func() *test {
			testTx := ts.NewTestTransaction(true)

			return &test{
				name: "ok - Basic block with transaction and tagged data payload - TransactionsIncludedBlockTopic",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
						isPollingTarget: true,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.Block)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.Block)),
				preSubscribeFunc: func() {
					// we need to add the block to the nodebridge, so that it is available
					// for the TransactionsIncludedBlockTopic
					ts.MockAddBlock(testTx.BlockID, testTx.Block)

					// we also need to add the first output to the nodebridge, so that it is available.
					// this is also used by the TransactionsIncludedBlockTopic to get the blockID of the block containing the transaction of that output
					ts.MockAddOutput(testTx.OutputID, testTx.Output)
				},
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromTransaction(testTx.BlockID, testTx.Transaction),
						},
					})
				},
			}
		}(),

		// ok - Basic block with tagged data - TopicBlockMetadata
		func() *test {
			blockMetadataResponse := &api.BlockMetadataResponse{
				BlockID:            tpkg.RandBlockID(),
				BlockState:         api.BlockStateAccepted,
				BlockFailureReason: api.BlockFailureNone,
				TransactionMetadata: &api.TransactionMetadataResponse{
					TransactionID:            tpkg.RandTransactionID(),
					TransactionState:         api.TransactionStateFailed,
					TransactionFailureReason: api.TxFailureBICInputInvalid,
				},
			}

			return &test{
				name: "ok - Basic block with tagged data - TopicBlockMetadata",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicBlockMetadata(blockMetadataResponse.BlockID),
						isPollingTarget: true,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.TopicBlockMetadataAccepted,
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(blockMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(blockMetadataResponse)),
				preSubscribeFunc: func() {
					ts.MockAddBlockMetadata(blockMetadataResponse.BlockID, blockMetadataResponse)
				},
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedBlock(lo.PanicOnErr(inx.WrapBlockMetadata(blockMetadataResponse)))
				},
			}
		}(),

		// ok - Basic block with tagged data - TopicBlockMetadataAccepted
		func() *test {
			blockMetadataResponse := &api.BlockMetadataResponse{
				BlockID:            tpkg.RandBlockID(),
				BlockState:         api.BlockStateAccepted,
				BlockFailureReason: api.BlockFailureNone,
				TransactionMetadata: &api.TransactionMetadataResponse{
					TransactionID:            tpkg.RandTransactionID(),
					TransactionState:         api.TransactionStateFailed,
					TransactionFailureReason: api.TxFailureBICInputInvalid,
				},
			}

			return &test{
				name: "ok - Basic block with tagged data - TopicBlockMetadataAccepted",
				topics: []*testTopic{
					{
						topic:           mqtt.TopicBlockMetadataAccepted,
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicBlockMetadata(blockMetadataResponse.BlockID),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(blockMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(blockMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedBlock(lo.PanicOnErr(inx.WrapBlockMetadata(blockMetadataResponse)))
				},
			}
		}(),

		// ok - Basic block with tagged data - TopicBlockMetadataConfirmed
		func() *test {
			blockMetadataResponse := &api.BlockMetadataResponse{
				BlockID:            tpkg.RandBlockID(),
				BlockState:         api.BlockStateAccepted,
				BlockFailureReason: api.BlockFailureNone,
				TransactionMetadata: &api.TransactionMetadataResponse{
					TransactionID:            tpkg.RandTransactionID(),
					TransactionState:         api.TransactionStateFailed,
					TransactionFailureReason: api.TxFailureBICInputInvalid,
				},
			}

			return &test{
				name: "ok - Basic block with tagged data - TopicBlockMetadataConfirmed",
				topics: []*testTopic{
					{
						topic:           mqtt.TopicBlockMetadataConfirmed,
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicBlockMetadata(blockMetadataResponse.BlockID),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(blockMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(blockMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveConfirmedBlock(lo.PanicOnErr(inx.WrapBlockMetadata(blockMetadataResponse)))
				},
			}
		}(),

		// ok - TopicOutputs
		func() *test {
			testTx := ts.NewTestTransaction(true)

			return &test{
				name: "ok - TopicOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicOutput(testTx.OutputID),
						isPollingTarget: true,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				preSubscribeFunc: func() {
					output := ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse)
					ts.MockAddOutput(testTx.OutputID, output)
				},
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicAccountOutputs
		func() *test {
			accountOutput := tpkg.RandOutput(iotago.OutputAccount).(*iotago.AccountOutput)
			testTx := ts.NewTestTransaction(true, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				accountOutput,
			}))

			return &test{
				name: "ok - TopicAccountOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicAccountOutputs(accountOutput.AccountID, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicAnchorOutputs
		func() *test {
			anchorOutput := tpkg.RandOutput(iotago.OutputAnchor).(*iotago.AnchorOutput)
			testTx := ts.NewTestTransaction(true, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				anchorOutput,
			}))

			return &test{
				name: "ok - TopicAnchorOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicAnchorOutputs(anchorOutput.AnchorID, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionStateController, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionGovernor, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicFoundryOutputs
		func() *test {
			foundryOutput := tpkg.RandOutput(iotago.OutputFoundry).(*iotago.FoundryOutput)
			testTx := ts.NewTestTransaction(true, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				foundryOutput,
			}))

			return &test{
				name: "ok - TopicFoundryOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicFoundryOutputs(lo.PanicOnErr(foundryOutput.FoundryID())),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionImmutableAccount, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicNFTOutputs
		func() *test {
			nftOutput := tpkg.RandOutput(iotago.OutputNFT).(*iotago.NFTOutput)
			testTx := ts.NewTestTransaction(true, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				nftOutput,
			}))

			return &test{
				name: "ok - TopicNFTOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicNFTOutputs(nftOutput.NFTID, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicDelegationOutputs
		func() *test {
			delegationOutput := tpkg.RandOutput(iotago.OutputDelegation).(*iotago.DelegationOutput)
			testTx := ts.NewTestTransaction(true, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				delegationOutput,
			}))

			return &test{
				name: "ok - TopicDelegationOutputs",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicDelegationOutputs(delegationOutput.DelegationID),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.OwnerAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicOutputsByUnlockConditionAndAddress - Address/StorageReturn/Expiration
		func() *test {
			unlockAddress := tpkg.RandEd25519Address()
			returnAddress := tpkg.RandEd25519Address()

			basicOutput := &iotago.BasicOutput{
				Amount: 1337,
				Mana:   1337,
				UnlockConditions: iotago.BasicOutputUnlockConditions{
					&iotago.AddressUnlockCondition{
						Address: unlockAddress,
					},
					&iotago.StorageDepositReturnUnlockCondition{
						ReturnAddress: returnAddress,
						Amount:        1337,
					},
					&iotago.ExpirationUnlockCondition{
						ReturnAddress: returnAddress,
						Slot:          1337,
					},
				},
				Features: iotago.BasicOutputFeatures{},
			}

			testTx := ts.NewTestTransaction(false, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				basicOutput,
			}))

			return &test{
				name: "ok - TopicOutputsByUnlockConditionAndAddress - Address/StorageReturn/Expiration",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, unlockAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, returnAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, unlockAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionStorageReturn, returnAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionExpiration, returnAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicOutputsByUnlockConditionAndAddress - StateController/Governor
		func() *test {
			anchorOutput := tpkg.RandOutput(iotago.OutputAnchor).(*iotago.AnchorOutput)

			// we want to have different addresses for the state controller and governor to check the "any" topic
			anchorOutput.UnlockConditionSet().GovernorAddress().Address = tpkg.RandAddress(iotago.AddressEd25519)

			stateControllerAddress := anchorOutput.UnlockConditionSet().StateControllerAddress().Address
			governorAddress := anchorOutput.UnlockConditionSet().GovernorAddress().Address

			testTx := ts.NewTestTransaction(false, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				anchorOutput,
			}))

			return &test{
				name: "ok - TopicOutputsByUnlockConditionAndAddress - StateController/Governor",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, stateControllerAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, governorAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionStateController, stateControllerAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionGovernor, governorAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicAnchorOutputs(anchorOutput.AnchorID, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),

		// ok - TopicOutputsByUnlockConditionAndAddress - ImmutableAccount
		func() *test {
			foundryOutput := tpkg.RandOutput(iotago.OutputFoundry).(*iotago.FoundryOutput)
			immutableAccountAddress := foundryOutput.UnlockConditionSet().ImmutableAccount().Address

			testTx := ts.NewTestTransaction(false, tpkg.WithOutputs(iotago.Outputs[iotago.TxEssenceOutput]{
				foundryOutput,
			}))

			return &test{
				name: "ok - TopicOutputsByUnlockConditionAndAddress - ImmutableAccount",
				topics: []*testTopic{
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, immutableAccountAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
					{
						topic:           mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionImmutableAccount, immutableAccountAddress, ts.API().ProtocolParameters().Bech32HRP()),
						isPollingTarget: false,
						isEventTarget:   true,
					},
				},
				topicsIgnore: []string{
					mqtt.GetTopicTransactionsIncludedBlock(testTx.TransactionID),
					mqtt.GetTopicOutput(testTx.ConsumedOutputID),
					mqtt.GetTopicOutput(testTx.OutputID),
					mqtt.GetTopicFoundryOutputs(lo.PanicOnErr(foundryOutput.FoundryID())),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAny, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
					mqtt.GetTopicOutputsByUnlockConditionAndAddress(mqtt.UnlockConditionAddress, testTx.SenderAddress, ts.API().ProtocolParameters().Bech32HRP()),
				},
				jsonTarget: lo.PanicOnErr(ts.API().JSONEncode(testTx.OutputWithMetadataResponse)),
				rawTarget:  lo.PanicOnErr(ts.API().Encode(testTx.OutputWithMetadataResponse)),
				postSubscribeFunc: func() {
					ts.ReceiveAcceptedTransaction(&nodebridge.AcceptedTransaction{
						API:           ts.API(),
						Slot:          testTx.BlockID.Slot(),
						TransactionID: testTx.TransactionID,
						// the consumed input
						Consumed: []*nodebridge.Output{
							ts.NewSpentNodeBridgeOutputFromTransaction(tpkg.RandBlockID(), testTx.ConsumedOutputCreationTransaction, testTx.BlockID.Slot(), testTx.TransactionID),
						},
						// the created output
						Created: []*nodebridge.Output{
							ts.NewNodeBridgeOutputFromOutputWithMetadata(testTx.OutputWithMetadataResponse),
						},
					})
				},
			}
		}(),
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			ts.Reset()

			ts.MQTTClientConnect("client1")
			defer ts.MQTTClientDisconnect("client1")

			topicsReceivedLock := sync.Mutex{}
			topicsReceived := make(map[string]struct{})
			subscriptionDone := false

			// wrap it in a function to avoid the topic variable to be overwritten by the loop
			subscribeToMQTTTopic := func(topic *testTopic) {
				ts.MQTTSubscribe("client1", topic.topic, func(topicName string, payloadBytes []byte) {
					if !subscriptionDone && !topic.isPollingTarget {
						// if the subscription is not done and it is not a polled target, we don't want to receive it.
						return
					}

					if subscriptionDone && !topic.isEventTarget {
						// if the subscription is done and it is not an event target, we don't want to receive it.
						return
					}

					require.Equal(t, topic.topic, topicName, "topic mismatch")
					require.Equal(t, test.jsonTarget, payloadBytes, "JSON payload mismatch")

					topicsReceivedLock.Lock()
					defer topicsReceivedLock.Unlock()

					topicsReceived[topicName] = struct{}{}
				})

				// also subscribe to the raw topics
				topicNameRaw := topic.topic + "/raw"
				ts.MQTTSubscribe("client1", topicNameRaw, func(topicName string, payloadBytes []byte) {
					if !subscriptionDone && !topic.isPollingTarget {
						// if the subscription is not done and it is not a polled target, we don't want to receive it.
						return
					}

					if subscriptionDone && !topic.isEventTarget {
						// if the subscription is done and it is not an event target, we don't want to receive it.
						return
					}

					require.Equal(t, topicNameRaw, topicName, "topic mismatch")
					require.Equal(t, test.rawTarget, payloadBytes, "raw payload mismatch")

					topicsReceivedLock.Lock()
					defer topicsReceivedLock.Unlock()
					topicsReceived[topicName] = struct{}{}
				})
			}

			// check that we don't receive topics we don't subscribe to
			receivedTopicsLock := sync.Mutex{}
			receivedTopics := make(map[string]int)
			ts.MQTTSetHasSubscribersCallback(func(topicName string) {
				for _, ignoredTopic := range test.topicsIgnore {
					if topicName == ignoredTopic {
						return
					}
					if topicName == ignoredTopic+"/raw" {
						return
					}
				}

				receivedTopicsLock.Lock()
				defer receivedTopicsLock.Unlock()
				receivedTopics[topicName]++
			})

			// collect all topics for later comparison with the received topics
			collectedTopics := lo.Reduce(test.topics, func(collectedTopics map[string]int, topic *testTopic) map[string]int {
				if topic.isPollingTarget {
					// we need to add 2, because we subscribe to the json and the raw topic, and the initial "polled message"
					// on subscribe also checks for subscribers on the each other topic
					collectedTopics[topic.topic] += 2
					collectedTopics[topic.topic+"/raw"] += 2
				}
				if topic.isEventTarget {
					collectedTopics[topic.topic]++
					collectedTopics[topic.topic+"/raw"]++
				}

				return collectedTopics
			}, make(map[string]int))

			// this step can be used to receive "polled" topics (inject the payload before subscribing)
			if test.preSubscribeFunc != nil {
				test.preSubscribeFunc()
			}

			// subscribe to the topics
			for _, testTopic := range test.topics {
				subscribeToMQTTTopic(testTopic)
			}

			// unfortunately we need to wait a bit here, because the MQTT broker is running in a separate goroutine
			time.Sleep(50 * time.Millisecond)

			// everything we receive now is "event driven"
			subscriptionDone = true

			// this step can be used to receive "event driven" topics
			if test.postSubscribeFunc != nil {
				test.postSubscribeFunc()

				// unfortunately we need to wait a bit here, because the MQTT broker is running in a separate goroutine
				time.Sleep(50 * time.Millisecond)
			}

			// check if all topics were received
			for _, testTopic := range test.topics {
				require.Containsf(t, topicsReceived, testTopic.topic, "topic not received: %s", testTopic.topic)
				require.Containsf(t, topicsReceived, testTopic.topic+"/raw", "topic not received: %s", testTopic.topic+"/raw")
			}

			// check that we don't receive topics we don't subscribe to
			topicsReceivedLock.Lock()
			defer topicsReceivedLock.Unlock()

			for topic, count := range receivedTopics {
				if _, ok := collectedTopics[topic]; !ok {
					require.Failf(t, "received topic that was not subscribed to", "topic: %s, count: %d", topic, count)
				}

				require.Equalf(t, collectedTopics[topic], count, "topic count mismatch: %s", topic)
			}
		})
	}
}
