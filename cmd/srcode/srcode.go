package main

import (
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"strings"
)

var (
	// Version is updated by goreleaser using ldflags
	// https://goreleaser.com/environment/
	version = "dev"
)

func main() {
	app := cli.App{
		Name:    "srcode",
		Usage:   "Source code manager",
		Version: version,
		Description: `
srcode is a tool that help developers to manage their codebase
in an effective & productive way.

The codebase relies on a special git repository to track the changes
(new / deleted) projects and synchronize the up-to-date configuration
remotely.`,
		Commands: []*cli.Command{
			{
				Name:      "init",
				Usage:     "Create an empty codebase",
				Action:    initCodebase,
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "remote",
						Usage: "The codebase remote",
					},
				},
				Description: `
This command creates an empty codebase - basically a .srcode directory with manifest file to track the codebase projects.`,
			},
			{
				Name:      "clone",
				Usage:     "Clone a codebase into a new directory",
				Action:    cloneCodebase,
				ArgsUsage: "<remote> [<path>]",
				Description: `
Clones a codebase into a newly created directory, and install (clone) the existing projects.`,
			},
			{
				Name:      "add",
				Usage:     "Add a project to the codebase",
				Action:    addProject,
				ArgsUsage: "<remote> [<path>]",
				Flags: []cli.Flag{
					&cli.StringSliceFlag{
						Name:  "git-config",
						Usage: "Git configuration to apply (format key=value)",
					},
				},
				Description: `
Add a project (git repository) to the current codebase.`,
			},
			{
				Name:   "sync",
				Usage:  "Synchronize the codebase with the linked remote",
				Action: syncCodebase,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete-removed",
						Usage: "Deleted removed projects",
					},
				},
				Description: `
Synchronize the codebase with the linked remote - i.e install & configure new project and remove removed ones,
while pushing the changes.`,
			},
			{
				Name:   "pwd",
				Usage:  "Print codebase working directory",
				Action: pwdCodebase,
				Description: `
Print the codebase working directory - i.e the working directory relative to the codebase root.`,
			},
			{
				Name:      "run",
				Usage:     "Run a codebase command",
				Action:    runCodebase,
				ArgsUsage: "<command>",
				Description: `
Run a command inside a codebase project.`,
			},
			{
				Name:   "ls",
				Usage:  "Display the codebase projects",
				Action: lsCodebase,
				Description: `
Display the codebase projects with their details.`,
			},
		},
		Authors: []*cli.Author{{
			Name:  "Aloïs Micard",
			Email: "alois@micard.lu",
		}},
		Action: runCodebase, // shortcut: use srcode <command> to execute a codebase command easily
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

	path := c.Args().Get(0)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		path = filepath.Join(cwd, path)
	}

	if _, err := codebase.DefaultProvider.Init(path, c.String("remote")); err != nil {
		return err
	}

	fmt.Printf("Successfully initialized new codebase at: %s\n", path)

	return nil
}

func cloneCodebase(c *cli.Context) error {
	if c.Args().Len() < 1 {
		return fmt.Errorf("correct usage: srcode clone <remote> [<path>]")
	}

	path := c.Args().Get(1)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		path = filepath.Join(cwd, path)
	}

	ch := make(chan codebase.ProjectEntry)

	// Use goroutine to have un-buffered channel
	go func() {
		for entry := range ch {
			fmt.Printf("Cloned %s -> /%s\n", entry.Project.Remote, entry.Path)
		}
	}()

	if _, err := codebase.DefaultProvider.Clone(c.Args().First(), path, ch); err != nil {
		return nil
	}

	fmt.Printf("Successfully cloned codebase from %s to: %s\n", c.Args().First(), path)

	return nil
}

func addProject(c *cli.Context) error {
	if c.Args().Len() < 1 {
		return fmt.Errorf("correct usage: srcode add <remote> [<path>]")
	}

	cb, err := openCodebase()
	if err != nil {
		return err
	}

	path := ""
	if arg := c.Args().Get(1); arg != "" {
		path = arg
	}

	if _, err := cb.Add(c.Args().First(), path, parseGitConfig(c.StringSlice("git-config"))); err != nil {
		return err
	}

	fmt.Printf("Successfully added %s to: %s\n", c.Args().First(), path)

	return nil
}

func syncCodebase(c *cli.Context) error {
	cb, err := openCodebase()
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
	cb, err := openCodebase()
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

	cb, err := openCodebase()
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

func lsCodebase(c *cli.Context) error {
	cb, err := openCodebase()
	if err != nil {
		return err
	}

	projects, err := cb.Projects()
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		fmt.Println("No projects found")
	}

	for path, project := range projects {
		fmt.Printf("/%s -> %s\n", path, project.Remote)
	}

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

func openCodebase() (codebase.Codebase, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cb, err := codebase.DefaultProvider.Open(cwd)
	if err != nil {
		return nil, err
	}

	return cb, nil
}
