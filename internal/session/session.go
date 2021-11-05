package session

import (
	"github.com/dgraph-io/ristretto"
	"github.com/google/uuid"
	"github.com/neicnordic/sda-download/internal/config"
	log "github.com/sirupsen/logrus"
)

// SessionCache stores dataset permission lists
var SessionCache *ristretto.Cache

// DatasetCache stores the dataset permissions
// and information whether this information has
// already been checked or not. This information
// can then be used to skip the time-costly
// authentication middleware
// DatasetCache==nil, session doesn't exist
// DatasetCache.Datasets==nil, session exists, user has no permissions (this case is not used in middleware.go)
// DatasetCache.Datasets==[]string{}, session exists, user has permissions
type DatasetCache struct {
	Datasets []string
}

// InitialiseSessionCache creates a cache manager that stores keys and values in memory
func InitialiseSessionCache() (*ristretto.Cache, error) {
	log.Debug("creating session cache")
	sessionCache, err := ristretto.NewCache(
		&ristretto.Config{
			NumCounters: 1e7,     // Num keys to track frequency of (10 M = max. 1 M users)
			MaxCost:     1 << 30, // Maximum cost of cache (1 GiB)
			BufferItems: 64,      // Number of keys per Get buffer
		},
	)
	if err != nil {
		log.Errorf("failed to create session cache, reason=%v", err)
		return nil, err
	}
	log.Debug("session cache created")
	return sessionCache, nil
}

// Get returns a value from cache at key
// DON'T USE DEBUG LOGS IN PRODUCTION, AS SESSION KEY WILL BE LOGGED
func Get(key string) ([]string, bool) {
	log.Debugf("get value from cache at key=%s", key)
	header, exists := SessionCache.Get(key)
	var cachedDatasets []string
	if header != nil {
		cachedDatasets = header.(DatasetCache).Datasets
	} else {
		cachedDatasets = nil
	}
	log.Debugf("cached at key=%s, exists=%t, datasets=%s", key, exists, cachedDatasets)
	return cachedDatasets, exists
}

// Set stores data to cache at key
// DON'T USE DEBUG LOGS IN PRODUCTION, AS SESSION KEY WILL BE LOGGED
func Set(key string, datasets []string) {
	log.Debugf("store to cache at key=%s", key)
	datasetCache := DatasetCache{
		Datasets: datasets,
	}
	SessionCache.SetWithTTL(key, datasetCache, 1, config.Config.Session.Expiration)
	log.Debugf("stored to cached at key=%s", key)
}

// NewSessionKey generates a session key used for storing
// dataset permissions, and checks that it doesn't already exist
func NewSessionKey() string {
	log.Debug("generating new session key")

	// Generate a new key until one is generated, which doesn't already exist
	var sessionKey string
	exists := true
	for exists {

		// Generate the key
		key := uuid.New().String()
		sessionKey = key

		// Check if the generated key already exists in the cache
		_, exists = Get(key)
	}

	log.Debug("new session key generated")
	return sessionKey
}
