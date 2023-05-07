package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"

	"ariga.io/gh-atlas/gen"
	"github.com/alecthomas/kong"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"github.com/pkg/browser"
)

func main() {
	c, err := gh.HTTPClient(nil)
	if err != nil {
		log.Fatal(err)
	}
	client := github.NewClient(c)
	ghClient := &githubClient{
		Git:          client.Git,
		Repositories: client.Repositories,
		Actions:      client.Actions,
		PullRequests: client.PullRequests,
	}
	currRepo, err := gh.CurrentRepository()
	if err != nil {
		log.Fatal(err)
	}
	ctx := kong.Parse(&cli, kong.BindTo(currRepo, (*repository.Repository)(nil)))
	err = ctx.Run(context.Background(), ghClient, currRepo)
	ctx.FatalIfErrorf(err)
}

// cli is the root command.
var cli struct {
	InitCI InitCiCmd `cmd:"" help:"Initialize a new Atlas CI configuration."`
}

// InitCiCmd is the command for initializing a new Atlas CI workflow.
type InitCiCmd struct {
	DirPath string `arg:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver  string `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory."`
	Token   string `required:"" short:"t" help:"(Required) Atlas authentication token."`
	Repo    string `short:"R" help:"GitHub repository owner/name, defaults to the current repository."`
}

func (i *InitCiCmd) Help() string {
	return `Example:
	  gh atlas init-ci --driver="mysql" --token=$ATLAS_CLOUD_TOKEN "dir/migrations"`
}

const (
	commitMsg = "Add Atlas CI configuration yaml to GitHub Workflows"
	prTitle   = "Add Atlas CI configuration"
)

// Run the init-ci command.
func (i *InitCiCmd) Run(ctx context.Context, client *githubClient, current repository.Repository) error {
	var (
		err        error
		branchName = "atlas-ci-" + randSeq(6)
		secretName = "ATLAS_CLOUD_TOKEN"
	)
	if i.Repo != "" {
		current, err = repository.Parse(i.Repo)
		if err != nil {
			return err
		}
	}
	repoData, _, err := client.Repositories.Get(ctx, current.Owner(), current.Name())
	if err != nil {
		return err
	}
	repo := NewRepository(ctx, client, current, repoData.GetDefaultBranch())
	if err = repo.SetSecret(secretName, i.Token); err != nil {
		return err
	}
	if err = repo.CheckoutNewBranch(branchName); err != nil {
		return err
	}
	cfg := &gen.Config{
		Path:          i.DirPath,
		Driver:        i.Driver,
		SecretName:    secretName,
		DefaultBranch: repo.defaultBranch,
	}
	if err = repo.AddAtlasYAML(cfg, branchName, commitMsg); err != nil {
		return err
	}
	link, err := repo.CreatePR(prTitle, branchName)
	if err != nil {
		return err
	}
	if err = browser.OpenURL(link); err != nil {
		fmt.Printf("Failed to open %s in browser: %v\n", link, err)
	}
	return nil
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
