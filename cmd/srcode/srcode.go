package main

import (
	"errors"
	"fmt"
	"github.com/creekorful/srcode/internal/codebase"
	"github.com/creekorful/srcode/internal/manifest"
	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v2"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
)

var (
	// Version is updated by goreleaser using ldflags
	// https://goreleaser.com/environment/
	version = "dev"

	errWrongInitUsage       = errors.New("correct usage: srcode init <path>")
	errWrongCloneUsage      = errors.New("correct usage: srcode clone <remote> [<path>]")
	errWrongAddProjectUsage = errors.New("correct usage: srcode add <remote> [<path>]")
	errWrongRunUsage        = errors.New("correct usage: srcode run <script>")
	errWrongBulkGitUsage    = errors.New("correct usage: srcode bulk-git <args>")
	errWrongScriptUsage     = errors.New("correct usage: srcode script <name> [<script>]")
	errWrongMvUsage         = errors.New("correct usage: srcode mv <src> <dst>")
	errWrongRmUsage         = errors.New("correct usage: srcode rm <path>")
	errWrongHookUsage       = errors.New("correct usage: srcode hook <script>")
)

func main() {
	app := app{
		codebaseProvider: codebase.DefaultProvider,
		writer:           os.Stdout,
	}

	if err := app.getCliApp().Run(os.Args); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

type app struct {
	codebaseProvider codebase.Provider
	writer           io.Writer
}

func (app *app) getCliApp() *cli.App {
	return &cli.App{
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
				Action:    app.initCodebase,
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "remote",
						Usage: "The codebase remote",
					},
					&cli.BoolFlag{
						Name:  "import",
						Usage: "Import existing repositories located in codebase",
					},
				},
				Description: `
This command creates an empty codebase - basically a .srcode directory with manifest file to track the codebase projects.
If invoked with --import, srcode will search for existing Git repositories and import them as codebase projects.

Examples

- Initialize a codebase with specific remote:
  $ srcode init --remote git@github.com:creekorful/dot-srcode.git /path/to/custom/directory

- Initialize a codebase with existing projects:
  $ srcode init --import --remote git@github.com:creekorful/dot-srcode.git /path/to/custom/directory`,
			},
			{
				Name:      "clone",
				Usage:     "Clone a codebase into a new directory",
				Action:    app.cloneCodebase,
				ArgsUsage: "<remote> [<path>]",
				Description: `
Clones a codebase into a newly created directory, and install (clone) the existing projects.

Examples

- Clone a codebase into specific directory:
  $ srcode clone git@github.com:creekorful/dot-srcode.git /path/to/custom/directory`,
			},
			{
				Name:      "add",
				Usage:     "Add a project to the codebase",
				Action:    app.addProject,
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
  $ srcode add --git-config user.email=alois@micard.lu --git-config commit.gpgsign=true git@github.com:darkspot-org/bathyscaphe.git Darkspot/bathyscaphe`,
			},
			{
				Name:   "sync",
				Usage:  "Synchronize the codebase with the linked remote",
				Action: app.syncCodebase,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete-removed",
						Usage: "Deleted removed projects",
					},
				},
				Description: `
Synchronize the codebase with the linked remote - i.e install & configure new project and remove removed ones,
while pushing the changes.

Examples

- Synchronize with remote and delete removed projects:
  $ srcode sync --delete-removed`,
			},
			{
				Name:   "pwd",
				Usage:  "Print codebase working directory",
				Action: app.pwd,
				Description: `
Print the codebase working directory - i.e the working directory relative to the codebase root.`,
			},
			{
				Name:      "run",
				Usage:     "Run a codebase script",
				Action:    app.runScript,
				ArgsUsage: "<script>",
				Description: `
Run a script inside a codebase project.

Examples

- Execute a script named lint:
  $ srcode run lint
  $ srcode lint`,
			},
			{
				Name:   "ls",
				Usage:  "Display the codebase projects",
				Action: app.lsProjects,
				Description: `
Display the codebase projects with their details.`,
			},
			{
				Name:      "bulk-git",
				Usage:     "Execute a git command over all projects",
				Action:    app.bulkGit,
				ArgsUsage: "<args>",
				Description: `
Execute a git command in bulk (over all codebase projects).

Examples

- Update all repositories to their latest changes:
  $ srcode bulk-git pull --rebase`,
			},
			{
				Name:      "script",
				Usage:     "Add a codebase script",
				Action:    app.setScript,
				ArgsUsage: "<name> [<script>]",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "global",
						Usage: "If true make the script global",
					},
				},
				Description: `
Add a script to the codebase, either at global level (--global) or
at project level.

Examples

- Create a global go-test script:
  $ srcode script --global go-test go test -v ./...

- Link a project local test script to the previously defined global alias:
  $ srcode script test @go-test

- Create a project local test script, and edit it using $EDITOR:
  $ srcode script test

Now you can use 'srcode run test' or 'srcode test' to execute the script
from project directory.`,
			},
			{
				Name:      "mv",
				Usage:     "Move a project",
				Action:    app.mvProject,
				ArgsUsage: "<src> <dst>",
				Description: `
Move a project inside the codebase. This will move the physical folder
as well as update the manifest to reflect the changes.

Examples

- Move project from Personal/super-project to OldStuff/super-project-42:
  $ srcode mv Personal/super-project OldStuff/super-project-42`,
			},
			{
				Name:      "rm",
				Usage:     "Remove a project",
				Action:    app.rmProject,
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "delete",
						Usage: "If true delete the project from disk",
					},
				},
				Description: `
Remove a codebase project. This will remove it from the manifest and
optionally from the disk if --delete is provided.

Examples

- Remove a project located at Contributing/Test and remove from disk too:
  $ srcode rm --delete Contributing/Test`,
			},
			{
				Name:      "hook",
				Usage:     "Set project Git hook",
				Action:    app.hook,
				ArgsUsage: "<script>",
				Description: `
Set the Git hook of current project. This will lookup for the script with
given name, and copy the content to the .git/hooks folder.

Examples

- Set Git hook of current project:
  $ srcode hook lint`,
			},
		},
		Authors: []*cli.Author{{
			Name:  "Aloïs Micard",
			Email: "alois@micard.lu",
		}},
		Action: app.runScript, // shortcut: use srcode <script> to execute a codebase script easily
	}
}

func (app *app) initCodebase(c *cli.Context) error {
	if c.NArg() != 1 {
		return errWrongInitUsage
	}

	path := c.Args().Get(0)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}

		path = filepath.Join(cwd, path)
	}

	cb, err := app.codebaseProvider.Init(path, c.String("remote"), c.Bool("import"))
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "Successfully initialized new codebase at: %s\n", path)

	projects, err := cb.Projects()
	if err != nil {
		return err
	}

	for path, project := range projects {
		_, _ = fmt.Fprintf(app.writer, "Imported %s -> /%s\n", project.Project.Remote, path)
	}

	return nil
}

func (app *app) cloneCodebase(c *cli.Context) error {
	if c.NArg() < 1 {
		return errWrongCloneUsage
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

	wg.Add(1)
	go func() {
		for entry := range ch {
			_, _ = fmt.Fprintf(app.writer, "Cloned %s -> /%s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	_, err := app.codebaseProvider.Clone(c.Args().First(), path, ch)

	wg.Wait()

	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "Successfully cloned codebase from %s to: %s\n", c.Args().First(), path)

	return nil
}

func (app *app) addProject(c *cli.Context) error {
	if c.NArg() < 1 {
		return errWrongAddProjectUsage
	}

	cb, err := app.openCodebase()
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

	// little display trick
	path = "/" + path

	_, _ = fmt.Fprintf(app.writer, "Successfully added %s to: %s\n", c.Args().First(), path)

	return nil
}

func (app *app) syncCodebase(c *cli.Context) error {
	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	wg := sync.WaitGroup{}

	addedChan := make(chan codebase.ProjectEntry)
	wg.Add(1)
	go func() {
		for entry := range addedChan {
			_, _ = fmt.Fprintf(app.writer, "[+] %s -> %s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	deletedChan := make(chan codebase.ProjectEntry)
	wg.Add(1)
	go func() {
		for entry := range deletedChan {
			_, _ = fmt.Fprintf(app.writer, "[-] %s -> %s\n", entry.Project.Remote, entry.Path)
		}
		wg.Done()
	}()

	err = cb.Sync(c.Bool("delete-removed"), addedChan, deletedChan)

	wg.Wait()

	if err != nil {
		return err
	}

	_, _ = fmt.Fprintln(app.writer, "Successfully synchronized codebase")

	return nil
}

func (app *app) pwd(c *cli.Context) error {
	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "/%s\n", cb.LocalPath())

	return nil
}

func (app *app) runScript(c *cli.Context) error {
	if !c.Args().Present() {
		return errWrongRunUsage
	}

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	return cb.Run(c.Args().First(), app.writer)
}

func (app *app) lsProjects(c *cli.Context) error {
	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	projects, err := cb.Projects()
	if err != nil {
		return err
	}

	if len(projects) == 0 {
		_, _ = fmt.Fprintln(app.writer, "No projects in codebase")
		_, _ = fmt.Fprintln(app.writer, "Tips: add a project using `srcode add git@github.com:darkspot-org/bathyscaphe.git Darkspot/bathyscaphe`")
		return nil
	}

	// Sort keys to have re-producible output
	var keys []string
	for path := range projects {
		keys = append(keys, path)
	}
	sort.Strings(keys)

	table := tablewriter.NewWriter(app.writer)
	table.SetHeader([]string{"Remote", "Path", "Branch"})
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_CENTER})
	table.SetBorder(false)

	dirtStyle := color.New(color.Italic, color.FgHiYellow)
	for _, path := range keys {
		project := projects[path]

		branch, err := project.Repository.Head()
		if err != nil {
			return err
		}

		dirty, err := project.Repository.IsDirty()
		if err != nil {
			return err
		}

		values := []string{project.Project.Remote, "/" + path}
		if dirty {
			values = append(values, dirtStyle.Sprint(branch+"(*)"))
		} else {
			values = append(values, branch)
		}

		table.Append(values)
	}

	table.Render()

	return nil
}

func (app *app) bulkGit(c *cli.Context) error {
	if c.NArg() < 1 {
		return errWrongBulkGitUsage
	}

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	if err := cb.BulkGIT(c.Args().Slice(), app.writer); err != nil {
		return err
	}

	return nil
}

func (app *app) setScript(c *cli.Context) error {
	if c.NArg() < 1 {
		return errWrongScriptUsage
	}

	isGlobal := c.Bool("global")

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	man, err := cb.Manifest()
	if err != nil {
		return err
	}

	project, exist := man.Projects[cb.LocalPath()]

	// Make sure there's a project at current path (if adding local script)
	if !exist && !isGlobal {
		return manifest.ErrNoProjectFound
	}

	var script []string

	if c.NArg() >= 2 {
		// script provided directly trough CLI
		script = []string{strings.Join(c.Args().Tail(), " ")}
	} else {
		// get previous script definition
		var previousScript []string

		if isGlobal {
			previousScript = man.Scripts[c.Args().First()]
		} else {
			previousScript = project.Scripts[c.Args().First()]
		}

		// otherwise open $EDITOR and read input
		val, err := captureInputFromEditor(previousScript)
		if err != nil {
			return err
		}

		script = val

		if reflect.DeepEqual(script, previousScript) || len(script) == 0 {
			return nil // nothing to do
		}
	}

	return cb.SetScript(c.Args().First(), script, isGlobal)
}

func (app *app) mvProject(c *cli.Context) error {
	if c.NArg() < 2 {
		return errWrongMvUsage
	}

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	if err := cb.MoveProject(c.Args().First(), c.Args().Get(1)); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "Successfully moved from %s to %s\n", c.Args().First(), c.Args().Get(1))

	return nil
}

func (app *app) rmProject(c *cli.Context) error {
	if c.NArg() != 1 {
		return errWrongRmUsage
	}

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	if err := cb.RmProject(c.Args().First(), c.Bool("delete")); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "Successfully deleted %s\n", c.Args().First())

	return nil
}

func (app *app) hook(c *cli.Context) error {
	if c.NArg() != 1 {
		return errWrongHookUsage
	}

	cb, err := app.openCodebase()
	if err != nil {
		return err
	}

	if err := cb.SetHook(c.Args().First()); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(app.writer, "Successfully applied hook `%s` to /%s\n", c.Args().First(), cb.LocalPath())

	return nil
}

func (app *app) openCodebase() (codebase.Codebase, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cb, err := app.codebaseProvider.Open(cwd)
	if err != nil {
		return nil, err
	}

	return cb, nil
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

func captureInputFromEditor(initialContent []string) ([]string, error) {
	file, err := ioutil.TempFile(os.TempDir(), "*")
	if err != nil {
		return nil, err
	}
	path := file.Name()

	defer os.Remove(path)

	// write initial content if any
	if len(initialContent) > 0 {
		if _, err := io.WriteString(file, strings.Join(initialContent, "\n")); err != nil {
			return nil, err
		}
	}

	if err := file.Close(); err != nil {
		return nil, err
	}

	// lookup default editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim" // ;D
	}

	// open the editor on the temporary file
	cmd := exec.Command(editor, path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return strings.Split(strings.TrimSuffix(string(b), "\n"), "\n"), nil
}
