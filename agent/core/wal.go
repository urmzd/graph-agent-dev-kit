package core

// TxID uniquely identifies a WAL transaction.
type TxID string

// TxOpKind describes the type of operation in a transaction.
type TxOpKind string

const (
	TxOpAddNode       TxOpKind = "add_node"
	TxOpUpdateNode    TxOpKind = "update_node"
	TxOpSetBranch     TxOpKind = "set_branch"
	TxOpAddChild      TxOpKind = "add_child"
	TxOpAddCheckpoint TxOpKind = "add_checkpoint"
)

// TxOp is a single operation within a WAL transaction.
type TxOp struct {
	Kind       TxOpKind
	NodeID     NodeID
	ParentID   NodeID
	Node       *Node
	BranchID   BranchID
	TipID      NodeID
	Checkpoint *Checkpoint
}

// WAL provides write-ahead logging for atomic tree mutations.
type WAL interface {
	Begin() (TxID, error)
	Append(txID TxID, op TxOp) error
	Commit(txID TxID) error
	Abort(txID TxID) error
	Recover() ([]TxID, error)
	Replay(txID TxID) ([]TxOp, error)
}
