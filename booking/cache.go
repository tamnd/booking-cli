package booking

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"time"
)

// cache.go is a small on-disk response cache keyed by URL, so a crawl that
// revisits a landing page or a property does not refetch it. Entries are plain
// files under CacheDir named by the hash of the URL; freshness is the file mtime
// against CacheTTL. NoCache bypasses it; Refresh ignores hits but still writes.

// cacheKey is the on-disk filename for a URL.
func cacheKey(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return hex.EncodeToString(sum[:]) + ".html"
}

// cacheGet returns the cached body for a URL when caching is on, the entry
// exists, and it is within ttl. It returns nil otherwise.
func (c *Client) cacheGet(rawURL string) []byte {
	if c.cfg.NoCache || c.cfg.Refresh || c.cfg.CacheDir == "" {
		return nil
	}
	path := filepath.Join(c.cfg.CacheDir, cacheKey(rawURL))
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	ttl := c.cfg.CacheTTL
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	if time.Since(info.ModTime()) > ttl {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return b
}

// cachePut stores a body for a URL when caching is on. Write failures are
// ignored: the cache is an optimization, not a system of record.
func (c *Client) cachePut(rawURL string, body []byte) {
	if c.cfg.NoCache || c.cfg.CacheDir == "" {
		return
	}
	if err := os.MkdirAll(c.cfg.CacheDir, 0o755); err != nil {
		return
	}
	_ = os.WriteFile(filepath.Join(c.cfg.CacheDir, cacheKey(rawURL)), body, 0o644)
}
