package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestLocalRoundTripAndDelete(t *testing.T) {
	store, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	source := []byte("qutcraft-media")
	written, err := store.Put(context.Background(), "org-1/asset.txt", bytes.NewReader(source), "text/plain")
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	if written != int64(len(source)) {
		t.Fatalf("written = %d, want %d", written, len(source))
	}

	reader, err := store.Open(context.Background(), "org-1/asset.txt")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	actual, err := io.ReadAll(reader)
	_ = reader.Close()
	if err != nil || !bytes.Equal(actual, source) {
		t.Fatalf("read = %q, %v", actual, err)
	}

	if err := store.Delete(context.Background(), "org-1/asset.txt"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Open(context.Background(), "org-1/asset.txt"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Open after Delete = %v, want ErrNotFound", err)
	}
}

func TestLocalRejectsTraversalAndOutsideAbsolutePath(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	outside := filepath.Join(filepath.Dir(root), "outside.txt")
	for _, key := range []string{"../outside.txt", outside} {
		if _, err := store.Put(context.Background(), key, bytes.NewReader([]byte("unsafe")), "text/plain"); err == nil {
			t.Fatalf("Put(%q) should reject storage escape", key)
		}
	}
	if _, err := os.Stat(outside); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("outside file unexpectedly exists: %v", err)
	}
}

func TestLocalReadsLegacyAbsolutePathWithinRoot(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocal(root)
	if err != nil {
		t.Fatalf("NewLocal: %v", err)
	}
	legacyPath := filepath.Join(root, "legacy.png")
	if err := os.WriteFile(legacyPath, []byte("legacy"), 0o640); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	reader, err := store.Open(context.Background(), legacyPath)
	if err != nil {
		t.Fatalf("Open legacy path: %v", err)
	}
	_ = reader.Close()
}
