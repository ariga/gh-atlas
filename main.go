package main

import (
	"fmt"
	"math/rand"

	"github.com/alecthomas/kong"
	"github.com/pkg/browser"
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
	DirPath string `arg:"" optional:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver  string `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory."`
	Token   string `short:"t" help:"Atlas authentication token."`
}

func (i *InitCiCmd) Help() string {
	return `Example:
	gh atlas init-ci --token=$ATLAS_CLOUD_TOKEN
	gh atlas init-ci --token=$ATLAS_CLOUD_TOKEN --driver="mysql" "dir/migrations"`
}

const (
	commitMsg = "Add Atlas CI configuration yaml to GitHub Workflows"
	prTitle   = "Add Atlas CI configuration"
)

// Run the init-ci command.
func (i *InitCiCmd) Run() error {
	repo, err := NewRepository()
	if err != nil {
		return err
	}
	// if dir path is not defined we need to ask for the path and the driver
	if i.DirPath == "" {
		paths, err := repo.MigrationDirectories()
		if err != nil {
			return err
		}
		if len(paths) == 0 {
			return fmt.Errorf("no migration directories found in the repository")
		}
		i.DirPath, err = ask("choose migration directory", paths)
		if err != nil {
			return err
		}
		i.Driver, err = ask("choose driver", []string{"mysql", "postgres", "mariadb", "sqlite"})
		if err != nil {
			return err
		}
	}
	branchName := "atlas-ci-" + randSeq(6)
	if err = repo.CheckoutNewBranch(branchName); err != nil {
		return err
	}
	if err = repo.AddAtlasYAML(i.DirPath, i.Driver, branchName, commitMsg); err != nil {
		return err
	}
	link, err := repo.CreatePR(prTitle, branchName)
	if err != nil {
		return err
	}
	return browser.OpenURL(link)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
