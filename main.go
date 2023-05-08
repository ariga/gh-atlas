package main

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	opts := []kong.Option{
		kong.BindTo(context.Background(), (*context.Context)(nil)),
		kong.BindTo(currRepo, (*repository.Repository)(nil)),
	}
	ctx := kong.Parse(&cli, opts...)
	err = ctx.Run(context.Background(), ghClient, currRepo)
	ctx.FatalIfErrorf(err)
}

// cli is the root command.
var cli struct {
	InitCI InitCiCmd `cmd:"" help:"Initialize a new Atlas CI configuration."`
}

// InitCiCmd is the command for initializing a new Atlas CI workflow.
type InitCiCmd struct {
	DirPath string        `arg:"" optional:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver  string        `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory (mysql,postgres,mariadb,sqlite)."`
	Token   string        `short:"t" help:"Atlas authentication token."`
	Repo    string        `short:"R" help:"GitHub repository owner/name, defaults to the current repository."`
	stdin   io.ReadCloser `hidden:""`
}

func (i *InitCiCmd) Help() string {
	return `Examples:
	gh atlas init-ci
	gh atlas init-ci --token=$ATLAS_CLOUD_TOKEN
	gh atlas init-ci --token=$ATLAS_CLOUD_TOKEN --driver="mysql" "dir/migrations"`
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
	repo := NewRepository(client, current, repoData.GetDefaultBranch())
	dirs, err := repo.MigrationDirectories(ctx)
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		return errors.New("no migration directories found in the repository")
	}
	if err = i.setParams(dirs); err != nil {
		return err
	}
	if err = repo.SetSecret(ctx, secretName, i.Token); err != nil {
		return err
	}
	if err = repo.CheckoutNewBranch(ctx, branchName); err != nil {
		return err
	}
	cfg := &gen.Config{
		Path:          i.DirPath,
		Driver:        i.Driver,
		SecretName:    secretName,
		DefaultBranch: repo.defaultBranch,
	}
	if err = repo.AddAtlasYAML(ctx, cfg, branchName, commitMsg); err != nil {
		return err
	}
	link, err := repo.CreatePR(ctx, prTitle, branchName)
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
