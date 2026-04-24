package executor

import (
	"os"
	"path/filepath"
	"testing"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

func TestSaveAndLoadCurrentRelease(t *testing.T) {
	root := t.TempDir()
	release := NewReleaseRecord(root, filepath.Join(root, "bundle"), &deployartifact.Bundle{
		Provider: "azure",
		Files:    map[string]string{},
	})
	release.Status = "succeeded"

	if err := SaveRelease(root, release); err != nil {
		t.Fatalf("SaveRelease returned error: %v", err)
	}
	current, err := LoadCurrentRelease(root)
	if err != nil {
		t.Fatalf("LoadCurrentRelease returned error: %v", err)
	}
	if current.ID != release.ID {
		t.Fatalf("unexpected current release id: got %q want %q", current.ID, release.ID)
	}
}

func TestMaterializeReleaseBundleWritesStoredFiles(t *testing.T) {
	root := t.TempDir()
	release := &ReleaseRecord{
		ID:       "20260423T000002Z",
		Provider: "azure",
		BundleFiles: map[string]string{
			"deploy/azure/deploy.sh":  "echo ok",
			"deploy/azure/main.bicep": "resource foo 'Microsoft.Resources/deployments@2024-01-01' = {}",
		},
	}

	bundleRoot, bundle, err := MaterializeReleaseBundle(root, release)
	if err != nil {
		t.Fatalf("MaterializeReleaseBundle returned error: %v", err)
	}
	if bundle.Provider != "azure" {
		t.Fatalf("unexpected bundle provider: %q", bundle.Provider)
	}
	for _, relativePath := range []string{"deploy/azure/deploy.sh", "deploy/azure/main.bicep"} {
		if _, err := os.Stat(filepath.Join(bundleRoot, relativePath)); err != nil {
			t.Fatalf("expected materialized file %s: %v", relativePath, err)
		}
	}
}
