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
				Name:      "init",
				Usage:     "Initialize a new codebase",
				Action:    initCodebase,
				ArgsUsage: "<path>",
			},
			{
				Name:      "clone",
				Usage:     "Clone an existing codebase",
				Action:    cloneCodebase,
				ArgsUsage: "<remote> [<path>]",
			},
			{
				Name:      "add",
				Usage:     "Add a project",
				Action:    addProject,
				ArgsUsage: "<remote> [<path>]",
			},
			{
				Name:   "sync",
				Usage:  "Synchronize the codebase",
				Action: syncCodebase,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func initCodebase(c *cli.Context) error {
	if c.Args().Len() != 1 {
		return fmt.Errorf("correct usage: srcode init <path>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	if _, err := codebase.DefaultProvider.Init(filepath.Join(cwd, c.Args().First())); err != nil {
		return err
	}

	fmt.Printf("Successfully initialized new codebase at: %s\n", c.Args().First())

	return nil
}

func cloneCodebase(c *cli.Context) error {
	if c.Args().Len() < 1 {
		return fmt.Errorf("correct usage: srcode clone <remote> [<path>]")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := cwd
	if arg := c.Args().Get(1); arg != "" {
		path = filepath.Join(path, arg)
	}

	if _, err := codebase.DefaultProvider.Clone(c.Args().First(), path); err != nil {
		return nil
	}

	fmt.Printf("Successfully cloned codebase from %s to: %s\n", c.Args().First(), path)

	return nil
}

func addProject(c *cli.Context) error {
	if c.Args().Len() < 1 {
		return fmt.Errorf("correct usage: srcode add <remote> [<path>]")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	path := ""
	if arg := c.Args().Get(1); arg != "" {
		path = arg
	}

	cb, err := codebase.DefaultProvider.Open(cwd)
	if err != nil {
		return err
	}

	if err := cb.Add(c.Args().First(), path); err != nil {
		return err
	}

	fmt.Printf("Successfully added %s to: %s\n", c.Args().First(), path)

	return nil
}

func syncCodebase(c *cli.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cb, err := codebase.DefaultProvider.Open(cwd)
	if err != nil {
		return err
	}

	added, removed, err := cb.Sync()
	if err != nil {
		return err
	}

	fmt.Println("Successfully synchronized codebase")

	for path, project := range added {
		fmt.Printf("[+] %s -> %s\n", project.Remote, path)
	}

	for path, project := range removed {
		fmt.Printf("[-] %s -> %s\n", project.Remote, path)
	}

	return nil
}
