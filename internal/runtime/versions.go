package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const cacheTTL = 24 * time.Hour

// ==================== Cache ====================

var cacheMu sync.Mutex

type versionCache struct {
	FetchedAt map[string]int64      `json:"fetched_at"` // sdkType -> unix timestamp
	Versions  map[string][]string   `json:"versions"`
}

// legacyFetchedAt is used for backward-compatible deserialization of old cache format.
type legacyFetchedAt struct {
	FetchedAt int64              `json:"fetched_at"`
	Versions  map[string][]string `json:"versions"`
}

func cachePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".else-toolbox", "versions_cache.json")
}

func loadCache() *versionCache {
	data, err := os.ReadFile(cachePath())
	if err != nil {
		return nil
	}

	// Try new format first
	var c versionCache
	if err := json.Unmarshal(data, &c); err == nil {
		if c.FetchedAt == nil {
			c.FetchedAt = make(map[string]int64)
		}
		if c.Versions == nil {
			c.Versions = make(map[string][]string)
		}
		return &c
	}

	// Fallback: try legacy format (single FetchedAt timestamp)
	var legacy legacyFetchedAt
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil
	}
	c = versionCache{
		FetchedAt: make(map[string]int64),
		Versions:  legacy.Versions,
	}
	if c.Versions == nil {
		c.Versions = make(map[string][]string)
	}
	// Migrate: assign legacy timestamp to all known SDK types
	if legacy.FetchedAt > 0 {
		for key := range c.Versions {
			c.FetchedAt[key] = legacy.FetchedAt
		}
	}
	return &c
}

func saveCache(c *versionCache) {
	os.MkdirAll(filepath.Dir(cachePath()), 0700)
	data, _ := json.MarshalIndent(c, "", "  ")
	os.WriteFile(cachePath(), data, 0644)
}

func getCached(sdkType string) []string {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	c := loadCache()
	if c == nil {
		return nil
	}
	ts, ok := c.FetchedAt[sdkType]
	if !ok {
		return nil
	}
	age := time.Since(time.Unix(ts, 0))
	if age < cacheTTL {
		return c.Versions[sdkType]
	}
	// Expired, but keep as fallback
	return nil
}

func putCached(sdkType string, versions []string) {
	cacheMu.Lock()
	defer cacheMu.Unlock()

	c := loadCache()
	if c == nil {
		c = &versionCache{
			FetchedAt: make(map[string]int64),
			Versions:  make(map[string][]string),
		}
	}
	if c.FetchedAt == nil {
		c.FetchedAt = make(map[string]int64)
	}
	c.Versions[sdkType] = versions
	c.FetchedAt[sdkType] = time.Now().Unix()
	saveCache(c)
}

// ==================== Fetch ====================

func FetchAvailable(sdkType SDKType, force bool) []string {
	key := string(sdkType)
	isForce := force

	// Non-forced: try cache first
	if !isForce {
		if cached := getCached(key); cached != nil {
			// Cache hit — return immediately, refresh in background
			go func() {
				if fresh := fetchFromRegistry(sdkType); len(fresh) > 0 {
					putCached(key, fresh)
				}
			}()
			return cached
		}
	}

	// Forced or cache miss: fetch from upstream (blocking)
	versions := fetchFromRegistry(sdkType)

	if len(versions) > 0 {
		putCached(key, versions)
	} else if !isForce {
		// API failed on cache miss, fallback to expired cache
		cacheMu.Lock()
		cached := loadCache()
		cacheMu.Unlock()
		if cached != nil {
			return cached.Versions[key]
		}
	}

	return versions
}

func fetchFromRegistry(sdkType SDKType) []string {
	reg := GetRegistry(sdkType)
	if reg == nil || reg.Fetcher == nil {
		return nil
	}
	return reg.Fetcher()
}

// ==================== Go ====================

type goRelease struct {
	Version string `json:"version"`
	Stable  bool   `json:"stable"`
}

func fetchGoVersions() []string {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://go.dev/dl/?mode=json&include=all")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var releases []goRelease
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil
	}

	versions := make([]string, 0)
	for _, r := range releases {
		if r.Stable && r.Version != "" {
			versions = append(versions, r.Version)
		}
	}
	sortStringVersions(versions)
	return versions
}

// ==================== Node.js ====================

type nodeDistEntry struct {
	Version string      `json:"version"`
	LTS     interface{} `json:"lts"`
}

func fetchNodeVersions() []string {
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get("https://nodejs.org/dist/index.json")
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var entries []nodeDistEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil
	}

	versions := make([]string, 0)
	for _, e := range entries {
		if e.LTS != nil && e.LTS != false && e.Version != "" {
			versions = append(versions, e.Version)
		}
	}
	sortStringVersions(versions)
	return versions
}

// ==================== Java (Adoptium) ====================

func fetchJavaVersions() []string {
	client := &http.Client{Timeout: 15 * time.Second}
	versions := make(map[string]bool)

	for _, major := range []int{8, 11, 17, 21, 22, 23, 24} {
		url := fmt.Sprintf(
			"https://api.adoptium.net/v3/assets/feature_releases/%d/ga?page_size=1",
			major,
		)
		resp, err := client.Get(url)
		if err != nil {
			continue
		}

		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			continue
		}

		var assets []struct {
			Version struct {
				Major  int    `json:"major"`
				Minor  int    `json:"minor"`
				Semver string `json:"semver"`
			} `json:"version"`
		}
		if err := json.Unmarshal(data, &assets); err != nil {
			continue
		}

		if len(assets) > 0 {
			semver := assets[0].Version.Semver
			if semver != "" {
				versions[semver] = true
			}
		}
	}

	result := make([]string, 0, len(versions))
	for v := range versions {
		result = append(result, v)
	}
	sortStringVersions(result)
	return result
}

// ==================== Helpers ====================

func sortStringVersions(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) > 0
	})
}
