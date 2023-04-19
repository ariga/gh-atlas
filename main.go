package main

import (
	"errors"

	"ariga.io/gh-atlas/internal/github"
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

const (
	branchName = "atlas-ci"
	commitMsg  = "Add Atlas CI configuration yaml to GitHub Workflows"
	prTitle    = "Add Atlas CI configuration"
)

// Run the init-ci command.
func (i *InitCiCmd) Run() error {
	repo, err := github.NewGitHubRepository()
	if err != nil {
		return err
	}
	isDirty, err := repo.IsDirty()
	if err != nil {
		return err
	}
	if isDirty {
		return errors.New("working tree is dirty, please commit or stash your changes")
	}
	if err = repo.SetAtlasToken(i.Token); err != nil {
		return err
	}
	if err = repo.CheckoutNewBranch(branchName); err != nil {
		return err
	}
	if err = repo.AddAtlasYaml(i.DirPath); err != nil {
		return err
	}
	if err = repo.CommitChanges(commitMsg); err != nil {
		return err
	}
	if err = repo.PushChanges(branchName); err != nil {
		return err
	}
	return repo.CreatePR(prTitle, branchName)
}
