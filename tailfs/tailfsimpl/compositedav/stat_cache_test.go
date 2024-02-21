// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package compositedav

import (
	"bytes"
	"testing"
	"time"

	"tailscale.com/tstest"
)

func TestStatCache(t *testing.T) {
	// Make sure we don't leak goroutines
	tstest.ResourceCheck(t)

	val := []byte("1")
	ttl := 1 * time.Second
	c := &StatCache{TTL: ttl}

	// set new stat
	c.set("file", 1, val)
	fetched := c.get("file", 1)
	if !bytes.Equal(fetched, val) {
		t.Errorf("want %q, got %q", val, fetched)
	}

	// fetch stat again, should still be cached
	fetched = c.get("file", 1)
	if !bytes.Equal(fetched, val) {
		t.Errorf("want %q, got %q", val, fetched)
	}

	// wait for cache to expire and refetch stat, should be empty now
	time.Sleep(ttl * 2)

	fetched = c.get("file", 1)
	if fetched != nil {
		t.Errorf("invalidate should have cleared cached value")
	}

	c.set("file", 1, val)
	// invalidate the cache and make sure nothing is returned
	c.invalidate()
	fetched = c.get("file", 1)
	if fetched != nil {
		t.Errorf("invalidate should have cleared cached value")
	}

	c.stop()
}
