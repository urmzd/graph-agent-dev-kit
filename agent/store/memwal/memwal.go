package memwal

import (
	"fmt"
	"sync"

	"github.com/urmzd/graph-agent-dev-kit/agent/core"
)

// txState tracks the state of an in-flight transaction.
type txState struct {
	ops       []core.TxOp
	committed bool
	aborted   bool
}

// WAL is a WAL implementation backed by an in-memory map.
// Suitable for testing; offers no crash durability.
type WAL struct {
	mu     sync.Mutex
	txns   map[core.TxID]*txState
	nextID uint64
}

// New creates a new in-memory WAL.
func New() *WAL {
	return &WAL{
		txns: make(map[core.TxID]*txState),
	}
}

func (w *WAL) Begin() (core.TxID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.nextID++
	id := core.TxID(fmt.Sprintf("tx-%d", w.nextID))
	w.txns[id] = &txState{}
	return id, nil
}

func (w *WAL) Append(txID core.TxID, op core.TxOp) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	tx, ok := w.txns[txID]
	if !ok {
		return fmt.Errorf("unknown transaction: %s", txID)
	}
	if tx.committed || tx.aborted {
		return fmt.Errorf("transaction %s is already finalized", txID)
	}
	tx.ops = append(tx.ops, op)
	return nil
}

func (w *WAL) Commit(txID core.TxID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	tx, ok := w.txns[txID]
	if !ok {
		return fmt.Errorf("unknown transaction: %s", txID)
	}
	if tx.aborted {
		return fmt.Errorf("transaction %s was aborted", txID)
	}
	tx.committed = true
	return nil
}

func (w *WAL) Abort(txID core.TxID) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	tx, ok := w.txns[txID]
	if !ok {
		return fmt.Errorf("unknown transaction: %s", txID)
	}
	tx.aborted = true
	return nil
}

func (w *WAL) Recover() ([]core.TxID, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var committed []core.TxID
	for id, tx := range w.txns {
		if tx.committed {
			committed = append(committed, id)
		}
	}
	return committed, nil
}

func (w *WAL) Replay(txID core.TxID) ([]core.TxOp, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	tx, ok := w.txns[txID]
	if !ok {
		return nil, fmt.Errorf("unknown transaction: %s", txID)
	}
	ops := make([]core.TxOp, len(tx.ops))
	copy(ops, tx.ops)
	return ops, nil
}
