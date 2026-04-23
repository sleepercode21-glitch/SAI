package utils

import (
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	existing := filepath.Join("..", "examples", "orders.sai")
	if !FileExists(existing) {
		t.Fatalf("expected %s to exist", existing)
	}

	if FileExists(filepath.Join("..", "examples", "missing.sai")) {
		t.Fatal("expected missing path to not exist")
	}
}

func TestResolveManifestPathDefaultsToSaiManifest(t *testing.T) {
	path, err := ResolveManifestPath("")
	if err != nil {
		t.Fatalf("ResolveManifestPath returned error: %v", err)
	}

	if filepath.Base(path) != "sai.sai" {
		t.Fatalf("unexpected default manifest name: %s", path)
	}
}
