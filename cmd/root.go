package cmd

import (
	"ariga.io/gh-atlas/cmd/github"
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

const (
	branchName = "gh-atlas-ci-init"
	commitMsg  = "Add atlas.yml"
	prTitle    = "Add atlas.yml to github actions"
)

func (i *InitCiCmd) Run() error {
	repo, err := github.NewGitHubRepository()
	if err != nil {
		return err
	}
	if err = repo.SetAtlasToken(i.Token); err != nil {
		return err
	}
	cleanup, err := repo.CloneRepo()
	if err != nil {
		return err
	}
	defer cleanup()
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

func Execute() {
	ctx := kong.Parse(&cli)
	err := ctx.Run(&cli)
	ctx.FatalIfErrorf(err)
}
