package executor

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

const (
	StateDirName    = ".sai-state"
	CurrentFileName = "current.json"
	HistoryDirName  = "history"
	LogsDirName     = "logs"
	BundlesDirName  = "bundles"
)

type ReleaseRecord struct {
	ID               string            `json:"id"`
	Provider         string            `json:"provider"`
	BundleRoot       string            `json:"bundle_root"`
	Status           string            `json:"status"`
	Operation        string            `json:"operation,omitempty"`
	RollbackTargetID string            `json:"rollback_target_id,omitempty"`
	StartedAt        time.Time         `json:"started_at"`
	FinishedAt       time.Time         `json:"finished_at"`
	Commands         []CommandSpec     `json:"commands"`
	Events           []ExecutionEvent  `json:"events,omitempty"`
	LogPath          string            `json:"log_path"`
	BundleFiles      map[string]string `json:"bundle_files,omitempty"`
}

type ExecutionEvent struct {
	Name      string    `json:"name"`
	Command   string    `json:"command,omitempty"`
	Status    string    `json:"status"`
	Detail    string    `json:"detail,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func EnsureStateLayout(root string) error {
	for _, path := range []string{
		filepath.Join(root, StateDirName),
		filepath.Join(root, StateDirName, HistoryDirName),
		filepath.Join(root, StateDirName, LogsDirName),
		filepath.Join(root, StateDirName, BundlesDirName),
	} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return err
		}
	}
	return nil
}

func NewReleaseRecord(root, bundleRoot string, bundle *deployartifact.Bundle) *ReleaseRecord {
	now := time.Now().UTC()
	id := now.Format("20060102T150405.000000000Z")
	return &ReleaseRecord{
		ID:          id,
		Provider:    bundle.Provider,
		BundleRoot:  bundleRoot,
		Status:      "pending",
		Operation:   "deploy",
		StartedAt:   now,
		LogPath:     filepath.Join(root, StateDirName, LogsDirName, id+".log"),
		BundleFiles: bundle.Files,
	}
}

func SaveRelease(root string, release *ReleaseRecord) error {
	if err := EnsureStateLayout(root); err != nil {
		return err
	}

	data, err := json.MarshalIndent(release, "", "  ")
	if err != nil {
		return err
	}
	historyPath := filepath.Join(root, StateDirName, HistoryDirName, release.ID+".json")
	if err := os.WriteFile(historyPath, data, 0o644); err != nil {
		return err
	}
	if release.Status == "succeeded" {
		if err := os.WriteFile(filepath.Join(root, StateDirName, CurrentFileName), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func LoadCurrentRelease(root string) (*ReleaseRecord, error) {
	return loadReleaseFile(filepath.Join(root, StateDirName, CurrentFileName))
}

func LoadLatestRelease(root string) (*ReleaseRecord, error) {
	dir := filepath.Join(root, StateDirName, HistoryDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no recorded releases")
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return loadReleaseFile(filepath.Join(dir, names[len(names)-1]))
}

func LoadReleaseByID(root, releaseID string) (*ReleaseRecord, error) {
	return loadReleaseFile(filepath.Join(root, StateDirName, HistoryDirName, releaseID+".json"))
}

func BundleFromRelease(release *ReleaseRecord) *deployartifact.Bundle {
	return &deployartifact.Bundle{
		Provider: release.Provider,
		Files:    release.BundleFiles,
	}
}

func MaterializeReleaseBundle(root string, release *ReleaseRecord) (string, *deployartifact.Bundle, error) {
	if len(release.BundleFiles) == 0 {
		return "", nil, fmt.Errorf("release %s does not contain bundle files", release.ID)
	}
	bundleRoot := filepath.Join(root, StateDirName, BundlesDirName, release.ID)
	bundle := BundleFromRelease(release)
	for path, content := range bundle.Files {
		absolutePath := filepath.Join(bundleRoot, path)
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
			return "", nil, err
		}
		if err := os.WriteFile(absolutePath, []byte(content), 0o644); err != nil {
			return "", nil, err
		}
	}
	return bundleRoot, bundle, nil
}

func loadReleaseFile(path string) (*ReleaseRecord, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var release ReleaseRecord
	if err := json.Unmarshal(data, &release); err != nil {
		return nil, err
	}
	return &release, nil
}
