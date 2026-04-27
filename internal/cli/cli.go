package cli

type CLI struct {
	View ViewCmd `cmd:"" default:"withargs" help:"Open a Projects v2 board in the terminal."`
}
