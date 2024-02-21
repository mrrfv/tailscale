// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package compositedav

import (
	"sync"
	"time"

	"github.com/jellydator/ttlcache/v3"
)

// StatCache provides a cache for directory listings and file metadata.
// Especially when used from the command-line, mapped WebDAV drives can
// generate repetitive requests for the same file metadata. This cache helps
// reduce the number of round-trips to the WebDAV server for such requests.
// This is similar to the DirectoryCacheLifetime setting of Windows' built-in
// SMB client, see
// https://learn.microsoft.com/en-us/previous-versions/windows/it-pro/windows-7/ff686200(v=ws.10)
type StatCache struct {
	TTL time.Duration

	// mu guards the below values.
	mu     sync.Mutex
	caches map[int]*ttlcache.Cache[string, []byte]
}

func (c *StatCache) get(name string, depth int) []byte {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.caches == nil {
		return nil
	}
	cache := c.caches[depth]
	if cache == nil {
		return nil
	}
	item := cache.Get(name)
	if item == nil {
		return nil
	}
	return item.Value()
}

func (c *StatCache) set(name string, depth int, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.caches == nil {
		c.caches = make(map[int]*ttlcache.Cache[string, []byte])
	}
	cache := c.caches[depth]
	if cache == nil {
		cache = ttlcache.New(
			ttlcache.WithTTL[string, []byte](c.TTL),
		)
		go cache.Start()
		c.caches[depth] = cache
	}
	cache.Set(name, value, ttlcache.DefaultTTL)
}

func (c *StatCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cache := range c.caches {
		cache.DeleteAll()
	}
}

func (c *StatCache) stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, cache := range c.caches {
		cache.Stop()
	}
}
