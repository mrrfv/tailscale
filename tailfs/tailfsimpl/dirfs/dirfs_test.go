// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

package dirfs

import (
	"context"
	"errors"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tailscale/xnet/webdav"
	"tailscale.com/tailfs/tailfsimpl/shared"
	"tailscale.com/tstest"
)

func TestStat(t *testing.T) {
	cfs, _, _, clock, close := createFileSystem(t)
	defer close()

	tests := []struct {
		label    string
		name     string
		expected fs.FileInfo
		err      error
	}{
		{
			label: "root folder",
			name:  "",
			expected: &shared.StaticFileInfo{
				Named:      "",
				Sized:      0,
				ModdedTime: clock.Now(),
				Dir:        true,
			},
		},
		{
			label: "static root folder",
			name:  "/domain",
			expected: &shared.StaticFileInfo{
				Named:      "domain",
				Sized:      0,
				ModdedTime: clock.Now(),
				Dir:        true,
			},
		},
		{
			label: "remote1",
			name:  "/domain/remote1",
			expected: &shared.StaticFileInfo{
				Named:      "remote1",
				Sized:      0,
				ModdedTime: clock.Now(),
				Dir:        true,
			},
		},
		{
			label: "remote2",
			name:  "/domain/remote2",
			expected: &shared.StaticFileInfo{
				Named:      "remote2",
				Sized:      0,
				ModdedTime: clock.Now(),
				Dir:        true,
			},
		},
		{
			label: "non-existent remote",
			name:  "remote3",
			err:   os.ErrNotExist,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			fi, err := cfs.Stat(ctx, test.name)
			if test.err != nil {
				if err == nil || !errors.Is(err, test.err) {
					t.Errorf("expected error: %v   got: %v", test.err, err)
				}
			} else {
				if err != nil {
					t.Errorf("unable to stat file: %v", err)
				} else {
					infosEqual(t, test.expected, fi)
				}
			}
		})
	}
}

func TestListDir(t *testing.T) {
	cfs, _, _, clock, close := createFileSystem(t)
	defer close()

	tests := []struct {
		label    string
		name     string
		expected []fs.FileInfo
		err      error
	}{
		{
			label: "root folder",
			name:  "",
			expected: []fs.FileInfo{
				&shared.StaticFileInfo{
					Named:      "domain",
					Sized:      0,
					ModdedTime: clock.Now(),
					Dir:        true,
				},
			},
		},
		{
			label: "static root folder",
			name:  "/domain",
			expected: []fs.FileInfo{
				&shared.StaticFileInfo{
					Named:      "remote1",
					Sized:      0,
					ModdedTime: clock.Now(),
					Dir:        true,
				},
				&shared.StaticFileInfo{
					Named:      "remote2",
					Sized:      0,
					ModdedTime: clock.Now(),
					Dir:        true,
				},
				&shared.StaticFileInfo{
					Named:      "remote4",
					Sized:      0,
					ModdedTime: clock.Now(),
					Dir:        true,
				},
			},
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			var infos []fs.FileInfo
			file, err := cfs.OpenFile(ctx, test.name, os.O_RDONLY, 0)
			if err == nil {
				defer file.Close()
				infos, err = file.Readdir(0)
			}
			if test.err != nil {
				if err == nil || !errors.Is(err, test.err) {
					t.Errorf("expected error: %v   got: %v", test.err, err)
				}
			} else {
				if err != nil {
					t.Errorf("unable to stat file: %v", err)
				} else {
					if len(infos) != len(test.expected) {
						t.Errorf("wrong number of file infos, want %d, got %d", len(test.expected), len(infos))
					} else {
						for i, expected := range test.expected {
							infosEqual(t, expected, infos[i])
						}
					}
				}
			}
		})
	}
}

func TestMkdir(t *testing.T) {
	fs, _, _, _, close := createFileSystem(t)
	defer close()

	tests := []struct {
		label string
		name  string
		perm  os.FileMode
		err   error
	}{
		{
			label: "attempt to create root folder",
			name:  "/",
		},
		{
			label: "attempt to create static root folder",
			name:  "/domain",
		},
		{
			label: "attempt to create remote",
			name:  "/domain/remote1",
		},
		{
			label: "attempt to create non-existent remote",
			name:  "/domain/remote3",
			err:   os.ErrPermission,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			err := fs.Mkdir(ctx, test.name, test.perm)
			if test.err != nil {
				if err == nil || !errors.Is(err, test.err) {
					t.Errorf("expected error: %v   got: %v", test.err, err)
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestRemoveAll(t *testing.T) {
	fs, _, _, _, close := createFileSystem(t)
	defer close()

	tests := []struct {
		label string
		name  string
		err   error
	}{
		{
			label: "attempt to remove root folder",
			name:  "/",
			err:   os.ErrPermission,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			err := fs.RemoveAll(ctx, test.name)
			if err == nil || !errors.Is(err, test.err) {
				t.Errorf("expected error: %v   got: %v", test.err, err)
			}
		})
	}
}

func TestRename(t *testing.T) {
	fs, _, _, _, close := createFileSystem(t)
	defer close()

	tests := []struct {
		label   string
		oldName string
		newName string
		err     error
	}{
		{
			label:   "attempt to move root folder",
			oldName: "/",
			newName: "/domain/remote2/copy.txt",
			err:     os.ErrPermission,
		},
	}

	ctx := context.Background()
	for _, test := range tests {
		t.Run(test.label, func(t *testing.T) {
			err := fs.Rename(ctx, test.oldName, test.newName)
			if err == nil || test.err.Error() != err.Error() {
				t.Errorf("expected error: %v   got: %v", test.err, err)
			}
		})
	}
}

func createFileSystem(t *testing.T) (webdav.FileSystem, string, string, *tstest.Clock, func()) {
	l1, dir1 := startRemote(t)
	l2, dir2 := startRemote(t)

	// Make some files, use perms 0666 as lowest common denominator that works
	// on both UNIX and Windows.
	err := os.WriteFile(filepath.Join(dir1, "file1.txt"), []byte("12345"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir2, "file2.txt"), []byte("54321"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	// make some directories
	err = os.Mkdir(filepath.Join(dir1, "dir1"), 0666)
	if err != nil {
		t.Fatal(err)
	}
	err = os.Mkdir(filepath.Join(dir2, "dir2"), 0666)
	if err != nil {
		t.Fatal(err)
	}

	clock := tstest.NewClock(tstest.ClockOpts{Start: time.Now()})
	fs := &FS{
		Clock:      clock,
		StaticRoot: "domain",
		Children: []*Child{
			{Name: "remote1"},
			{Name: "remote2"},
			{Name: "remote4"},
		},
	}

	return fs, dir1, dir2, clock, func() {
		defer l1.Close()
		defer os.RemoveAll(dir1)
		defer l2.Close()
		defer os.RemoveAll(dir2)
	}
}

func startRemote(t *testing.T) (net.Listener, string) {
	dir := t.TempDir()

	l, err := net.Listen("tcp", "127.0.0.1:")
	if err != nil {
		t.Fatal(err)
	}

	h := &webdav.Handler{
		FileSystem: webdav.Dir(dir),
		LockSystem: webdav.NewMemLS(),
	}

	s := &http.Server{Handler: h}
	go s.Serve(l)

	return l, dir
}

func infosEqual(t *testing.T, expected, actual fs.FileInfo) {
	t.Helper()
	if expected.Name() != actual.Name() {
		t.Errorf("expected name: %v   got: %v", expected.Name(), actual.Name())
	}
	if expected.Size() != actual.Size() {
		t.Errorf("expected Size: %v   got: %v", expected.Size(), actual.Size())
	}
	if !expected.ModTime().Truncate(time.Second).UTC().Equal(actual.ModTime().Truncate(time.Second).UTC()) {
		t.Errorf("expected ModTime: %v   got: %v", expected.ModTime(), actual.ModTime())
	}
	if expected.IsDir() != actual.IsDir() {
		t.Errorf("expected IsDir: %v   got: %v", expected.IsDir(), actual.IsDir())
	}
}
