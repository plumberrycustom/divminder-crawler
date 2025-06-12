package cache

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sirupsen/logrus"
)

// FileCache implements a simple file-based cache with TTL support
type FileCache struct {
	cacheDir string
	ttl      time.Duration
	logger   *logrus.Logger
}

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Data      interface{} `json:"data"`
	CreatedAt time.Time   `json:"createdAt"`
	ExpiresAt time.Time   `json:"expiresAt"`
	Key       string      `json:"key"`
}

// NewFileCache creates a new file-based cache
func NewFileCache(cacheDir string, ttl time.Duration) *FileCache {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		logger.Errorf("Failed to create cache directory %s: %v", cacheDir, err)
	}

	return &FileCache{
		cacheDir: cacheDir,
		ttl:      ttl,
		logger:   logger,
	}
}

// generateCacheKey creates a consistent cache key from input
func (fc *FileCache) generateCacheKey(prefix string, params ...string) string {
	// Combine all parameters into a single string
	combined := prefix
	for _, param := range params {
		combined += "_" + param
	}

	// Generate MD5 hash for consistent filename
	hash := md5.Sum([]byte(combined))
	return fmt.Sprintf("%s_%x.json", prefix, hash)
}

// getCacheFilePath returns the full path to a cache file
func (fc *FileCache) getCacheFilePath(key string) string {
	return filepath.Join(fc.cacheDir, key)
}

// Set stores data in the cache with TTL
func (fc *FileCache) Set(key string, data interface{}) error {
	now := time.Now()
	entry := CacheEntry{
		Data:      data,
		CreatedAt: now,
		ExpiresAt: now.Add(fc.ttl),
		Key:       key,
	}

	filePath := fc.getCacheFilePath(key)

	// Create the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create cache file %s: %w", filePath, err)
	}
	defer file.Close()

	// Encode to JSON
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(entry); err != nil {
		return fmt.Errorf("failed to encode cache entry: %w", err)
	}

	fc.logger.Debugf("Cached data with key: %s (expires: %s)", key, entry.ExpiresAt.Format(time.RFC3339))
	return nil
}

// Get retrieves data from the cache if it exists and hasn't expired
func (fc *FileCache) Get(key string, target interface{}) (bool, error) {
	filePath := fc.getCacheFilePath(key)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		fc.logger.Debugf("Cache miss: %s (file not found)", key)
		return false, nil
	}

	// Read the file
	file, err := os.Open(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to open cache file %s: %w", filePath, err)
	}
	defer file.Close()

	// Decode JSON
	var entry CacheEntry
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&entry); err != nil {
		fc.logger.Warnf("Failed to decode cache file %s, removing: %v", filePath, err)
		os.Remove(filePath)
		return false, nil
	}

	// Check if expired
	if time.Now().After(entry.ExpiresAt) {
		fc.logger.Debugf("Cache expired: %s (expired: %s)", key, entry.ExpiresAt.Format(time.RFC3339))
		os.Remove(filePath)
		return false, nil
	}

	// Convert data to target type
	dataBytes, err := json.Marshal(entry.Data)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cached data: %w", err)
	}

	if err := json.Unmarshal(dataBytes, target); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached data: %w", err)
	}

	fc.logger.Debugf("Cache hit: %s (created: %s)", key, entry.CreatedAt.Format(time.RFC3339))
	return true, nil
}

// Delete removes an item from the cache
func (fc *FileCache) Delete(key string) error {
	filePath := fc.getCacheFilePath(key)

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete cache file %s: %w", filePath, err)
	}

	fc.logger.Debugf("Deleted cache entry: %s", key)
	return nil
}

// Clear removes all cache files
func (fc *FileCache) Clear() error {
	entries, err := os.ReadDir(fc.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	deletedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			filePath := filepath.Join(fc.cacheDir, entry.Name())
			if err := os.Remove(filePath); err != nil {
				fc.logger.Warnf("Failed to delete cache file %s: %v", filePath, err)
			} else {
				deletedCount++
			}
		}
	}

	fc.logger.Infof("Cleared %d cache entries", deletedCount)
	return nil
}

// CleanExpired removes expired cache entries
func (fc *FileCache) CleanExpired() error {
	entries, err := os.ReadDir(fc.cacheDir)
	if err != nil {
		return fmt.Errorf("failed to read cache directory: %w", err)
	}

	now := time.Now()
	expiredCount := 0

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		filePath := filepath.Join(fc.cacheDir, entry.Name())

		// Read and check expiration
		file, err := os.Open(filePath)
		if err != nil {
			fc.logger.Warnf("Failed to open cache file %s: %v", filePath, err)
			continue
		}

		var cacheEntry CacheEntry
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&cacheEntry)
		file.Close()

		if err != nil {
			fc.logger.Warnf("Failed to decode cache file %s, removing: %v", filePath, err)
			os.Remove(filePath)
			expiredCount++
			continue
		}

		// Remove if expired
		if now.After(cacheEntry.ExpiresAt) {
			if err := os.Remove(filePath); err != nil {
				fc.logger.Warnf("Failed to delete expired cache file %s: %v", filePath, err)
			} else {
				expiredCount++
			}
		}
	}

	if expiredCount > 0 {
		fc.logger.Infof("Cleaned %d expired cache entries", expiredCount)
	}

	return nil
}

// GetStats returns cache statistics
func (fc *FileCache) GetStats() (map[string]interface{}, error) {
	entries, err := os.ReadDir(fc.cacheDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache directory: %w", err)
	}

	totalFiles := 0
	totalSize := int64(0)
	expiredFiles := 0
	now := time.Now()

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		totalFiles++

		// Get file size
		filePath := filepath.Join(fc.cacheDir, entry.Name())
		if info, err := entry.Info(); err == nil {
			totalSize += info.Size()
		}

		// Check if expired
		file, err := os.Open(filePath)
		if err != nil {
			continue
		}

		var cacheEntry CacheEntry
		decoder := json.NewDecoder(file)
		err = decoder.Decode(&cacheEntry)
		file.Close()

		if err == nil && now.After(cacheEntry.ExpiresAt) {
			expiredFiles++
		}
	}

	return map[string]interface{}{
		"totalEntries":   totalFiles,
		"totalSizeKB":    totalSize / 1024,
		"expiredEntries": expiredFiles,
		"cacheDir":       fc.cacheDir,
		"ttlHours":       fc.ttl.Hours(),
	}, nil
}

// ETFMetadataCache provides specialized caching for ETF metadata
type ETFMetadataCache struct {
	cache *FileCache
}

// NewETFMetadataCache creates a cache specifically for ETF metadata
func NewETFMetadataCache(cacheDir string, ttl time.Duration) *ETFMetadataCache {
	return &ETFMetadataCache{
		cache: NewFileCache(filepath.Join(cacheDir, "etf_metadata"), ttl),
	}
}

// GetETFMetadata retrieves cached ETF metadata
func (emc *ETFMetadataCache) GetETFMetadata(symbol string) (interface{}, bool, error) {
	key := emc.cache.generateCacheKey("etf_metadata", symbol)

	var metadata interface{}
	found, err := emc.cache.Get(key, &metadata)

	return metadata, found, err
}

// SetETFMetadata caches ETF metadata
func (emc *ETFMetadataCache) SetETFMetadata(symbol string, metadata interface{}) error {
	key := emc.cache.generateCacheKey("etf_metadata", symbol)
	return emc.cache.Set(key, metadata)
}

// InvalidateETF removes cached data for a specific ETF
func (emc *ETFMetadataCache) InvalidateETF(symbol string) error {
	key := emc.cache.generateCacheKey("etf_metadata", symbol)
	return emc.cache.Delete(key)
}

// CleanExpired removes expired ETF metadata entries
func (emc *ETFMetadataCache) CleanExpired() error {
	return emc.cache.CleanExpired()
}

// GetStats returns cache statistics
func (emc *ETFMetadataCache) GetStats() (map[string]interface{}, error) {
	return emc.cache.GetStats()
}
