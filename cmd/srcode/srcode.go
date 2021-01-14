package main

import (
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
)

func main() {
	app := cli.App{
		Name:    "srcode",
		Usage:   "Manage your codebase like a boss!",
		Version: "0.1.0",
		Authors: []*cli.Author{{
			Name:  "Aloïs Micard",
			Email: "alois@micard.lu",
		}},
		Commands: []*cli.Command{
			{
				Name:   "new",
				Usage:  "Create a new codebase",
				Action: newCodebase,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func newCodebase(c *cli.Context) error {
	if !c.Args().Present() {
		return fmt.Errorf("correct usage: srcode new <path>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if _, err := codebase.DefaultProvider.New(filepath.Join(cwd, c.Args().First())); err != nil {
		return err
	}

	fmt.Printf("Successfully created new codebase at: %s\n", c.Args().First())

	return nil
}
