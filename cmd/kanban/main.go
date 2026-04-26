package main

import (
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/shuntaka9576/kanban/internal/cli"
)

var (
	Version   = "dev"
	BuildDate = ""
)

func main() {
	versionString := Version
	if BuildDate != "" {
		versionString = fmt.Sprintf("%s (%s)", Version, BuildDate)
	}

	var root cli.CLI
	parser, err := kong.New(&root,
		kong.Name("gh-kanban"),
		kong.Description("GitHub Projects v2 TUI – run `gh kanban view -u USER -p TITLE` to open a board."),
		kong.UsageOnError(),
		kong.Vars{"version": versionString},
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx, err := parser.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := ctx.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
