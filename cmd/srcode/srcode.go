package main

import (
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
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
		Action: runCodebase, // shortcut: use srcode <command> to execute a codebase command easily
		Commands: []*cli.Command{
			{
				Name:      "init",
				Usage:     "Initialize a new codebase",
				Action:    initCodebase,
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "remote",
						Usage: "The codebase remote",
					},
				},
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
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "git-config",
						Usage: "Git configuration to apply (format key=value)",
					},
				},
			},
			{
				Name:   "sync",
				Usage:  "Synchronize the codebase",
				Action: syncCodebase,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete-removed",
						Usage: "Deleted removed projects",
					},
				},
			},
			{
				Name:   "pwd",
				Usage:  "Print codebase working directory",
				Action: pwdCodebase,
			},
			{
				Name:      "run",
				Usage:     "Run a codebase command",
				Action:    runCodebase,
				ArgsUsage: "<command>",
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

	if _, err := codebase.DefaultProvider.Init(filepath.Join(cwd, c.Args().First()), c.String("remote")); err != nil {
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

	if _, err := cb.Add(c.Args().First(), path, parseGitConfig(c.StringSlice("git-config"))); err != nil {
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

	added, removed, err := cb.Sync(c.Bool("delete-removed"))
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

func pwdCodebase(c *cli.Context) error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cb, err := codebase.DefaultProvider.Open(cwd)
	if err != nil {
		return err
	}

	fmt.Printf("/%s\n", cb.LocalPath())

	return nil
}

func runCodebase(c *cli.Context) error {
	if !c.Args().Present() {
		return fmt.Errorf("correct usage: srcode run <command>")
	}

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	cb, err := codebase.DefaultProvider.Open(cwd)
	if err != nil {
		return err
	}

	res, err := cb.Run(c.Args().First())
	if err != nil {
		return err
	}

	fmt.Printf("%s\n", res)

	return nil
}

func parseGitConfig(args []string) map[string]string {
	config := map[string]string{}

	for _, arg := range args {
		parts := strings.Split(arg, "=")
		if len(parts) == 2 {
			config[parts[0]] = parts[1]
		}
	}

	return config
}
