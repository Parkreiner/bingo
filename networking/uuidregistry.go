package networking

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type uuidRegistryEntry struct {
	lastUsed  time.Time
	ipAddress string
	id        uuid.UUID
}

// uuidRegistry maps each IP address to a different UUID
type uuidRegistry struct {
	entries map[string]uuidRegistryEntry
	mtx     *sync.Mutex
}

func newUUIDRegistry() uuidRegistry {
	return uuidRegistry{
		entries: make(map[string]uuidRegistryEntry),
		mtx:     &sync.Mutex{},
	}
}

func (ur uuidRegistry) upsertIPAddress(ipAddress string) uuid.UUID {
	ur.mtx.Lock()
	defer ur.mtx.Unlock()

	entry, ok := ur.entries[ipAddress]
	if ok {
		return entry.id
	}

	newID := uuid.New()
	ur.entries[ipAddress] = uuidRegistryEntry{
		id:        newID,
		ipAddress: ipAddress,
		lastUsed:  time.Now(),
	}

	return newID
}
