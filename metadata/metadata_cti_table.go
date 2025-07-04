package metadata

import (
	"sync"
)

// CTITable provides thread-safe CTI ↔ ID mapping
// and re-uses IDs that were freed via Remove.
type CTITable struct {
	mu      sync.RWMutex
	entity  map[string]int64 // CTI to ID
	freeIDs []int64          // LIFO stack of reusable IDs
	counter int64
}

// NewCTITable returns an empty table.
func NewCTITable() *CTITable {
	return &CTITable{
		entity: make(map[string]int64),
	}
}

// Add stores the entity and returns its numeric ID.
// If the entity already has a positive ID, we keep it;
// otherwise we draw from the free-list or allocate a fresh one.
func (t *CTITable) Add(cti string) int64 {
	t.mu.Lock()
	defer t.mu.Unlock()

	// If cti already has an ID, return it.
	if id, ok := t.entity[cti]; ok {
		return id
	}

	var id int64
	n := len(t.freeIDs)
	if n > 0 {
		id = t.freeIDs[n-1]
		t.freeIDs = t.freeIDs[:n-1]
	} else {
		t.counter++
		id = t.counter
	}

	t.entity[cti] = id
	return id
}

// Lookup returns (ID, true) if found, (0, false) otherwise.
func (t *CTITable) Lookup(cti string) (int64, bool) {
	t.mu.RLock()
	id, ok := t.entity[cti]
	t.mu.RUnlock()
	return id, ok
}

// Remove deletes cti and makes its ID available for reuse.
// It returns true if something was removed.
func (t *CTITable) Remove(cti string) bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	id, ok := t.entity[cti]
	if !ok {
		return false
	}
	delete(t.entity, cti)

	t.freeIDs = append(t.freeIDs, id)
	return true
}

func (t *CTITable) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.entity = make(map[string]int64)
	t.freeIDs = nil
	t.counter = 0
}

var GlobalCTITable *CTITable = NewCTITable()
