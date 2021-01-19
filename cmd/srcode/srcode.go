package main

import (
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
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
This command creates an empty codebase - basically a .srcode directory with manifest file to track the codebase projects.

Examples

- Initialize a codebase with specific remote:
  srcode init --remote git@github.com:creekorful/dot-srcode.git /path/to/custom/directory`,
			},
			{
				Name:      "clone",
				Usage:     "Clone a codebase into a new directory",
				Action:    cloneCodebase,
				ArgsUsage: "<remote> [<path>]",
				Description: `
Clones a codebase into a newly created directory, and install (clone) the existing projects.

Examples

- Clone a codebase into specific directory:
  srcode clone git@github.com:creekorful/dot-srcode.git /path/to/custom/directory`,
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
Add a project (git repository) to the current codebase.

Examples

- Add a project with custom git configuration:
  srcode add --git-config user.email=alois@micard.lu --git-config commit.gpgsign=true git@github.com:darkspot-org/bathyscaphe.git Darkspot/bathyscaphe`,
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
Run a command inside a codebase project.

Examples

- Execute a command named lint:
  srcode run lint
  srcode lint`,
			},
			{
				Name:   "ls",
				Usage:  "Display the codebase projects",
				Action: lsCodebase,
				Description: `
Display the codebase projects with their details.`,
			},
			{
				Name:      "bulk-git",
				Usage:     "Execute a git command over all projects",
				Action:    bulkGitCodebase,
				ArgsUsage: "<args>",
				Description: `
Execute a git command in bulk (over all codebase projects).

Examples

- Update all repositories to their latest changes:
  srcode bulk-git pull --rebase`,
			},
			{
				Name:      "set-cmd",
				Usage:     "Add a codebase command",
				Action:    setCmdCodebase,
				ArgsUsage: "<name> <command>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "global",
						Usage: "If true make the command global",
					},
				},
				Description: `
Add a command to the codebase, either at global level (--global) or
at project level.

Examples

- Create a global go-test command:
  srcode set-cmd --global go-test go test -v ./...

- Link a project local test command to the previously defined global alias:
  srcode set-cmd test @go-test

Now you can use 'srcode run test' or 'srcode test' to execute the command
from project directory.`,
			},
			{
				Name:      "mv",
				Usage:     "Move a project",
				Action:    mvCodebase,
				ArgsUsage: "<src> <dst>",
				Description: `
Move a project inside the codebase. This will move the physical folder
as well as update the manifest to reflect the changes.

Examples

- Move project from Personal/super-project to OldStuff/super-project-42:
  srcode mv Personal/super-project OldStuff/super-project-42`,
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
	if c.NArg() != 1 {
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
	if c.NArg() < 1 {
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

	wg := sync.WaitGroup{}

	// Use goroutine to have un-buffered channel
	ch := make(chan codebase.ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range ch {
			fmt.Printf("Cloned %s -> /%s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	_, err := codebase.DefaultProvider.Clone(c.Args().First(), path, ch)

	wg.Wait()

	if err != nil {
		return err
	}

	fmt.Printf("Successfully cloned codebase from %s to: %s\n", c.Args().First(), path)

	return nil
}

func addProject(c *cli.Context) error {
	if c.NArg() < 1 {
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

	wg := sync.WaitGroup{}

	addedChan := make(chan codebase.ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range addedChan {
			fmt.Printf("[+] %s -> %s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	deletedChan := make(chan codebase.ProjectEntry)
	go func() {
		wg.Add(1)
		for entry := range deletedChan {
			fmt.Printf("[-] %s -> %s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	err = cb.Sync(c.Bool("delete-removed"), addedChan, deletedChan)

	wg.Wait()

	if err != nil {
		return err
	}

	fmt.Println("Successfully synchronized codebase")

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

	// Sort keys to have re-producible output
	var keys []string
	for path := range projects {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Remote", "Path"})
	table.SetBorder(false)

	for _, path := range keys {
		project := projects[path]
		table.Append([]string{project.Remote, path})
	}

	table.Render()

	return nil
}

func bulkGitCodebase(c *cli.Context) error {
	cb, err := openCodebase()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	ch := make(chan string)
	go func() {
		wg.Add(1)
		for out := range ch {
			fmt.Printf("%s\n\n", out)
		}
		wg.Done()
	}()

	err = cb.BulkGIT(c.Args().Slice(), ch)

	wg.Wait()

	if err != nil {
		return err
	}

	return nil
}

func setCmdCodebase(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("correct usage: srcode set-cmd <name> <command>")
	}

	cb, err := openCodebase()
	if err != nil {
		return err
	}

	return cb.SetCommand(c.Args().First(), strings.Join(c.Args().Tail(), " "), c.Bool("global"))
}

func mvCodebase(c *cli.Context) error {
	if c.NArg() < 2 {
		return fmt.Errorf("correct usage: srcode mv <src> <dst>")
	}

	cb, err := openCodebase()
	if err != nil {
		return err
	}

	if err := cb.MoveProject(c.Args().First(), c.Args().Get(1)); err != nil {
		return err
	}

	fmt.Printf("Successfully moved from %s to %s\n", c.Args().First(), c.Args().Get(1))

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
