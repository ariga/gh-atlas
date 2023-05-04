package main

import (
	"errors"
	"log"
	"math/rand"

	"ariga.io/gh-atlas/gen"
	"github.com/alecthomas/kong"
	"github.com/cli/go-gh"
	"github.com/google/go-github/v49/github"
	"github.com/pkg/browser"
)

func main() {
	c, err := gh.HTTPClient(nil)
	if err != nil {
		log.Fatal(err)
	}
	client := github.NewClient(c)
	ctx := kong.Parse(&cli)
	err = ctx.Run(&githubClient{
		Git:          client.Git,
		Repositories: client.Repositories,
		Actions:      client.Actions,
		PullRequests: client.PullRequests,
	})
	ctx.FatalIfErrorf(err)
}

// cli is the root command.
var cli struct {
	InitCI InitCiCmd `cmd:"" help:"Initialize a new Atlas CI configuration."`
}

// InitCiCmd is the command for initializing a new Atlas CI workflow.
type InitCiCmd struct {
	DirPath string `arg:"" optional:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver  string `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory (mysql,postgres,mariadb,sqlite)."`
	Token   string `short:"t" help:"Atlas authentication token."`
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
func (i *InitCiCmd) Run(client *githubClient) error {
	var (
		branchName = "atlas-ci-" + randSeq(6)
		secretName = "ATLAS_CLOUD_TOKEN"
	)
	repo, err := NewRepository(client)
	if err != nil {
		return err
	}
	dirs, err := repo.MigrationDirectories()
	if err != nil {
		return err
	}
	if len(dirs) == 0 {
		return errors.New("no migration directories found in the repository")
	}
	if err = setParams(i, dirs); err != nil {
		return err
	}
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
