package main

import (
	"fmt"

	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}

// cli is the root command.
var cli struct {
	InitCI InitCiCmd `cmd:"" help:"Initialize a new Atlas CI configuration."`
}

// InitCiCmd is the command for initializing a new Atlas CI workflow.
type InitCiCmd struct {
	DirPath string `arg:"" type:"-path" help:"Path inside repository containing the migration files."`
	Token   string `short:"t" help:"Atlas authentication token."`
}

func (i *InitCiCmd) Help() string {
	return `Example:
	  atlas init-ci -t $ATLAS_CLOUD_TOKEN "dir/migrations"`
}

func (i *InitCiCmd) Run() error {
	fmt.Println("running command")
	return nil
}
