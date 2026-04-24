package cli

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/sleepercode/sai/executor"
)

type logsCommand struct{}

func NewLogsCommand() Command {
	return &logsCommand{}
}

func (c *logsCommand) Name() string {
	return "logs"
}

func (c *logsCommand) Description() string {
	return "Reserved for deployment and runtime log streaming"
}

func (c *logsCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	releaseID := fs.String("release", "", "Release identifier")
	stateRoot := fs.String("state-root", ".sai-out", "Root directory containing SAI execution state")
	_ = fs.Bool("follow", false, "Follow log output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	record, err := resolveRelease(*stateRoot, *releaseID)
	if err != nil {
		return err
	}
	fmt.Printf("release=%s provider=%s status=%s operation=%s\n", record.ID, record.Provider, record.Status, record.Operation)
	for _, event := range record.Events {
		fmt.Printf("[%s] %s: %s", event.Timestamp.Format(time.RFC3339), event.Name, event.Status)
		if event.Detail != "" {
			fmt.Printf(" (%s)", event.Detail)
		}
		fmt.Println()
	}
	data, err := os.ReadFile(record.LogPath)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func resolveRelease(root, releaseID string) (*executor.ReleaseRecord, error) {
	if releaseID == "" {
		record, err := executor.LoadCurrentRelease(root)
		if err == nil {
			return record, nil
		}
		return executor.LoadLatestRelease(root)
	}
	return executor.LoadReleaseByID(root, releaseID)
}
