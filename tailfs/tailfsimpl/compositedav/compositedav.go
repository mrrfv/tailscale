// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package compositedav provides an http.Handler that composes multiple WebDAV
// services into a single WebDAV service that presents each of them as its own
// folder.
package compositedav

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/tailscale/xnet/webdav"
	"tailscale.com/tailfs/tailfsimpl/dirfs"
	"tailscale.com/tailfs/tailfsimpl/shared"
	"tailscale.com/tstime"
	"tailscale.com/types/logger"
)

// Child is a child folder of this compositedav.
type Child struct {
	dirfs.Child

	// BaseURL is the base URL of the WebDAV service to which we'll proxy
	// requests for this Child. We will append the filename from the original
	// URL to this.
	BaseURL string

	// Transport (if specified) is the http transport to use when communicating
	// with this Child's WebDAV service.
	Transport http.RoundTripper

	rp       *httputil.ReverseProxy
	initOnce sync.Once
}

// CloseIdleConnections forcibly closes any idle connections on this Child's
// reverse proxy.
func (c *Child) CloseIdleConnections() {
	tr, ok := c.Transport.(*http.Transport)
	if ok {
		tr.CloseIdleConnections()
	}
}

func (c *Child) init() {
	c.initOnce.Do(func() {
		c.rp = &httputil.ReverseProxy{
			Transport: c.Transport,
			Rewrite:   func(r *httputil.ProxyRequest) {},
		}
	})
}

// Handler implements http.Handler by using a dirfs.FS for showing a virtual
// read-only folder that represents the Child WebDAV services as sub-folders
// and proxying all requests for resources on the children to those children
// via httputil.ReverseProxy instances.
type Handler struct {
	// Logf specifies a logging function to use.
	Logf logger.Logf

	// Clock, if specified, determines the current time. If not specified, we
	// default to time.Now().
	Clock tstime.Clock

	// StatCache is an optional cache for PROPFIND results.
	StatCache *StatCache

	// childrenMu guards children and staticRoot.
	childrenMu sync.RWMutex
	children   []*Child
	staticRoot string
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PROPFIND" {
		h.handlePROPFIND(w, r)
		return
	}

	if h.StatCache != nil && r.Method != "GET" {
		// TODO(oxtoacart): maybe be more selective about invalidating cache
		// If the user is performing a modification (e.g. PUT, MKDIR, etc),
		// we need to invalidat the StatCache to make sure we're not knowingly
		// showing stale stats.
		h.StatCache.invalidate()
	}

	mpl := h.maxPathLength(r)
	pathComponents := shared.CleanAndSplit(r.URL.Path)

	if len(pathComponents) >= mpl {
		h.delegate(pathComponents[mpl-1:], w, r)
		return
	}
	h.handle(w, r)
}

// handle handles the request locally using our dirfs.FS.
func (h *Handler) handle(w http.ResponseWriter, r *http.Request) {
	h.childrenMu.RLock()
	children := make([]*dirfs.Child, 0, len(h.children))
	for _, child := range h.children {
		children = append(children, &child.Child)
	}
	wh := &webdav.Handler{
		LockSystem: webdav.NewMemLS(),
		FileSystem: &dirfs.FS{
			Clock:      h.Clock,
			Children:   children,
			StaticRoot: h.staticRoot,
		},
	}
	h.childrenMu.RUnlock()

	wh.ServeHTTP(w, r)
}

// delegate sends the request to the Child WebDAV server.
func (h *Handler) delegate(pathComponents []string, w http.ResponseWriter, r *http.Request) string {
	childName := pathComponents[0]
	child := h.GetChild(childName)
	if child == nil {
		w.WriteHeader(http.StatusNotFound)
		return childName
	}
	u, err := url.Parse(child.BaseURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return childName
	}
	u.Path = path.Join(u.Path, shared.Join(pathComponents[1:]...))
	r.URL = u
	r.Host = u.Host
	child.rp.ServeHTTP(w, r)
	return childName
}

// SetChildren replaces the entire existing set of children with the given
// ones. If staticRoot is given, the children will appear with a subfolder
// bearing named <staticRoot>.
func (h *Handler) SetChildren(staticRoot string, children ...*Child) {
	for _, child := range children {
		child.init()
	}

	slices.SortFunc(children, func(a, b *Child) int {
		return strings.Compare(a.Name, b.Name)
	})

	h.childrenMu.Lock()
	oldChildren := children
	h.children = children
	h.staticRoot = staticRoot
	h.childrenMu.Unlock()

	for _, child := range oldChildren {
		child.CloseIdleConnections()
	}
}

// GetChild gets the Child identified by name, or nil if no matching child
// found.
func (h *Handler) GetChild(name string) *Child {
	h.childrenMu.RLock()
	defer h.childrenMu.RUnlock()

	_, child := h.findChildLocked(name)
	return child
}

// Close closes this Handler,including closing all idle connections on children
// and stopping the StatCache (if caching is enabled).
func (h *Handler) Close() {
	h.childrenMu.RLock()
	oldChildren := h.children
	h.children = nil
	h.childrenMu.RUnlock()

	for _, child := range oldChildren {
		child.CloseIdleConnections()
	}

	if h.StatCache != nil {
		h.StatCache.stop()
	}
}

func (h *Handler) findChildLocked(name string) (int, *Child) {
	var child *Child
	i, found := slices.BinarySearchFunc(h.children, name, func(child *Child, name string) int {
		return strings.Compare(child.Name, name)
	})
	if found {
		child = h.children[i]
	}
	return i, child
}

// func (h *Handler) logf(format string, args ...any) {
// 	if h.Logf != nil {
// 		h.Logf(format, args...)
// 		return
// 	}

// 	log.Printf(format, args...)
// }

// maxPathLength calculates the maximum length of a path that can be handled by
// this handler without delegating to a Child.
func (h *Handler) maxPathLength(r *http.Request) int {
	h.childrenMu.RLock()
	defer h.childrenMu.RUnlock()

	mpl := 1
	if h.staticRoot != "" {
		mpl++
	}
	return mpl
}
