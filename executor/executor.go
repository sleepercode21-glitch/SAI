package executor

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	deployartifact "github.com/sleepercode/sai/compiler/deploy"
)

type CommandSpec struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
	Dir  string   `json:"dir"`
}

type Result struct {
	BundleRoot string         `json:"bundle_root"`
	Provider   string         `json:"provider"`
	Commands   []CommandSpec  `json:"commands"`
	Release    *ReleaseRecord `json:"release,omitempty"`
}

type Commander interface {
	Run(ctx context.Context, spec CommandSpec) error
}

type OSCommander struct{}

func (OSCommander) Run(ctx context.Context, spec CommandSpec) error {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (OSCommander) RunWithWriters(ctx context.Context, spec CommandSpec, stdout io.Writer, stderr io.Writer) error {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

type Executor struct {
	Commander Commander
}

func New() *Executor {
	return &Executor{Commander: OSCommander{}}
}

func (e *Executor) ExecuteBundle(ctx context.Context, bundleRoot string, bundle *deployartifact.Bundle) (*Result, error) {
	if e.Commander == nil {
		e.Commander = OSCommander{}
	}

	commands, err := plannedCommands(bundleRoot, bundle)
	if err != nil {
		return nil, err
	}

	for _, command := range commands {
		if err := e.Commander.Run(ctx, command); err != nil {
			return nil, fmt.Errorf("execute %s failed: %w", command.Name, err)
		}
	}

	return &Result{
		BundleRoot: bundleRoot,
		Provider:   bundle.Provider,
		Commands:   commands,
	}, nil
}

func (e *Executor) ExecuteAndRecord(ctx context.Context, root string, bundleRoot string, bundle *deployartifact.Bundle) (*Result, error) {
	release := NewReleaseRecord(root, bundleRoot, bundle)
	return e.executeRecorded(ctx, root, bundleRoot, bundle, release)
}

func (e *Executor) RollbackToRelease(ctx context.Context, root string, target *ReleaseRecord) (*Result, error) {
	bundleRoot, bundle, err := MaterializeReleaseBundle(root, target)
	if err != nil {
		return nil, err
	}
	release := NewReleaseRecord(root, bundleRoot, bundle)
	release.Operation = "rollback"
	release.RollbackTargetID = target.ID
	return e.executeRecorded(ctx, root, bundleRoot, bundle, release)
}

func (e *Executor) executeRecorded(ctx context.Context, root string, bundleRoot string, bundle *deployartifact.Bundle, release *ReleaseRecord) (*Result, error) {
	if e.Commander == nil {
		e.Commander = OSCommander{}
	}
	if err := EnsureStateLayout(root); err != nil {
		return nil, err
	}
	commands, err := plannedCommands(bundleRoot, bundle)
	if err != nil {
		appendEvent(release, "plan-commands", "failed", "", err.Error())
		release.Status = "failed"
		release.FinishedAt = time.Now().UTC()
		_ = SaveRelease(root, release)
		return nil, err
	}
	release.Commands = commands
	appendEvent(release, "plan-commands", "ok", "", fmt.Sprintf("prepared %d command(s)", len(commands)))

	logFile, err := os.Create(release.LogPath)
	if err != nil {
		return nil, err
	}
	defer logFile.Close()

	writer := io.MultiWriter(os.Stdout, logFile)
	for _, command := range commands {
		fmt.Fprintf(writer, "$ %s %s\n", command.Name, formatArgs(command.Args))
		appendEvent(release, filepath.Base(command.Name), "running", commandString(command), fmt.Sprintf("working directory: %s", command.Dir))
		if err := SaveRelease(root, release); err != nil {
			return nil, err
		}
		if runner, ok := e.Commander.(interface {
			RunWithWriters(context.Context, CommandSpec, io.Writer, io.Writer) error
		}); ok {
			if err := runner.RunWithWriters(ctx, command, writer, writer); err != nil {
				appendEvent(release, filepath.Base(command.Name), "failed", commandString(command), err.Error())
				release.Status = "failed"
				release.FinishedAt = time.Now().UTC()
				_ = SaveRelease(root, release)
				return nil, fmt.Errorf("execute %s failed: %w", command.Name, err)
			}
			appendEvent(release, filepath.Base(command.Name), "succeeded", commandString(command), "command completed")
			continue
		}
		if err := e.Commander.Run(ctx, command); err != nil {
			appendEvent(release, filepath.Base(command.Name), "failed", commandString(command), err.Error())
			release.Status = "failed"
			release.FinishedAt = time.Now().UTC()
			_ = SaveRelease(root, release)
			return nil, fmt.Errorf("execute %s failed: %w", command.Name, err)
		}
		appendEvent(release, filepath.Base(command.Name), "succeeded", commandString(command), "command completed")
	}

	release.Status = "succeeded"
	release.FinishedAt = time.Now().UTC()
	appendEvent(release, "release", "succeeded", "", "deployment completed")
	if err := SaveRelease(root, release); err != nil {
		return nil, err
	}

	return &Result{
		BundleRoot: bundleRoot,
		Provider:   bundle.Provider,
		Commands:   commands,
		Release:    release,
	}, nil
}

func formatArgs(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return strings.Join(args, " ")
}

func commandString(spec CommandSpec) string {
	if len(spec.Args) == 0 {
		return spec.Name
	}
	return spec.Name + " " + formatArgs(spec.Args)
}

func appendEvent(release *ReleaseRecord, name string, status string, command string, detail string) {
	release.Events = append(release.Events, ExecutionEvent{
		Name:      name,
		Status:    status,
		Command:   command,
		Detail:    detail,
		Timestamp: time.Now().UTC(),
	})
}

func plannedCommands(bundleRoot string, bundle *deployartifact.Bundle) ([]CommandSpec, error) {
	switch bundle.Provider {
	case "azure":
		return []CommandSpec{
			{
				Name: "bash",
				Args: []string{filepath.Join(bundleRoot, "deploy/azure/deploy.sh")},
				Dir:  bundleRoot,
			},
		}, nil
	case "aws", "gcp":
		return []CommandSpec{
			{
				Name: "bash",
				Args: []string{filepath.Join(bundleRoot, "deploy/terraform/deploy.sh")},
				Dir:  bundleRoot,
			},
		}, nil
	default:
		return nil, fmt.Errorf("executor does not support provider %q", bundle.Provider)
	}
}
