package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

type initCommand struct{}

func NewInitCommand() Command {
	return &initCommand{}
}

func (c *initCommand) Name() string {
	return "init"
}

func (c *initCommand) Description() string {
	return "Create a starter sai.sai manifest"
}

func (c *initCommand) Run(args []string) error {
	fs := flag.NewFlagSet(c.Name(), flag.ContinueOnError)
	path := fs.String("path", "sai.sai", "Path to the manifest to create")
	force := fs.Bool("force", false, "Overwrite an existing manifest")
	if err := fs.Parse(args); err != nil {
		return err
	}

	absPath, err := filepath.Abs(*path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absPath); err == nil && !*force {
		return fmt.Errorf("manifest already exists at %s; use --force to overwrite", absPath)
	}

	const template = `app "orders" {
  users 5000
  budget 75usd
  env prod
}

service api {
  runtime node
  path "server"
  port 3000
  public http
  connects postgres
}

database postgres {
  type managed
  size small
}
`

	if err := os.WriteFile(absPath, []byte(template), 0o644); err != nil {
		return err
	}

	fmt.Printf("created %s\n", absPath)
	return nil
}
