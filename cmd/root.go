package cmd

import (
	"fmt"

	"github.com/alecthomas/kong"
)

var cli struct {
	InitCI InitCiCmd `cmd:"" help:"Initialize a new Atlas CI configuration."`
}

type InitCiCmd struct {
	DirPath string `arg:"" type:"-path" help:"Path inside repository to the directory containing the migration files."`
	Token   string `short:"t" help:"Atlas auth token."`
}

func (i *InitCiCmd) Help() string {
	return `Example:
	  atlas init-ci -t $ATLAS_CLOUD_TOKEN "dir/migrations"`
}

func (i *InitCiCmd) Run() error {
	fmt.Println("running command")
	return nil
}

func Execute() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
