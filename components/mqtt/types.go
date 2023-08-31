package mqtt

import (
	"encoding/json"

	inx "github.com/iotaledger/inx/go"
)

// commitemntInfoPayload defines the payload of the commitment latest and confirmed topics.
type commitemntInfoPayload struct {
	// The identifier of commitment.
	CommitmentID string `json:"commitmentId"`
	// The slot index of the commitment.
	CommitmentSlotIndex uint64 `json:"commitmentSlotIndex"`
}

// blockMetadataPayload defines the payload of the block metadata topic.
type blockMetadataPayload struct {
	// The hex encoded block ID of the block.
	BlockID string `json:"blockId"`
	// The state of the block.
	//nolint:nosnakecase // grpc uses underscores
	BlockState inx.BlockMetadata_BlockState `json:"blockState,omitempty"`
	// The reason why the block failed.
	BlockFailureReason inx.BlockMetadata_BlockFailureReason `json:"blockFailureReason,omitempty"`
	// The state of the transaction.
	TxState inx.BlockMetadata_TransactionState `json:"txState,omitempty"`
	// The reason why the transaction failed.
	TxFailureReason inx.BlockMetadata_TransactionFailureReason `json:"txFailureReason,omitempty"`
}

// outputMetadataPayload defines the metadata of an output.
type outputMetadataPayload struct {
	// The hex encoded block ID of the block.
	BlockID string `json:"blockId"`
	// The hex encoded transaction id from which this output originated.
	TransactionID string `json:"transactionId"`
	// The milestone index at which this output was spent.
	MilestoneIndexSpent uint32 `json:"milestoneIndexSpent,omitempty"`
	// The milestone timestamp this output was spent.
	MilestoneTimestampSpent uint32 `json:"milestoneTimestampSpent,omitempty"`
	// The transaction this output was spent with.
	TransactionIDSpent string `json:"transactionIdSpent,omitempty"`
	// The milestone index at which this output was booked into the ledger.
	MilestoneIndexBooked uint32 `json:"milestoneIndexBooked"`
	// The milestone timestamp this output was booked in the ledger.
	MilestoneTimestampBooked uint32 `json:"milestoneTimestampBooked"`
	// The ledger index at which this output was available at.
	LedgerIndex uint32 `json:"ledgerIndex"`
	// The index of the output.
	OutputIndex uint16 `json:"outputIndex"`
	// Whether this output is spent.
	Spent bool `json:"isSpent"`
}

// outputPayload defines the payload of the output topics.
type outputPayload struct {
	// The output in its serialized form.
	RawOutput *json.RawMessage `json:"output"`
}
