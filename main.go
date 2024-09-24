package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"

	"github.com/1lann/promptui"
	"github.com/alecthomas/kong"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"

	"ariga.io/gh-atlas/cloudapi"
	"ariga.io/gh-atlas/gen"
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
		kong.UsageOnError(),
	}
	ctx := kong.Parse(&cli, opts...)
	err = ctx.Run(context.Background(), ghClient, currRepo)
	ctx.FatalIfErrorf(err)
}

// cli is the root command.
var cli struct {
	InitAction InitActionCmd `cmd:"" help:"Initialize a new Atlas CI Action configuration."`
}

// InitActionCmd is the command for initializing a new Atlas CI workflow.
type InitActionCmd struct {
	From             string        `optional:"" help:"URL of the current schema state."`
	To               string        `optional:"" help:"URL of the desired schema state."`
	DirPath          string        `arg:"" optional:"" type:"-path" help:"Path inside repository containing the migration files."`
	Token            string        `short:"t" help:"Atlas authentication token."`
	Repo             string        `short:"R" help:"GitHub repository owner/name, defaults to the current repository."`
	ConfigPath       string        `optional:"" help:"Path to atlas.hcl configuration file."`
	ConfigEnv        string        `optional:"" help:"The environment to use from the Atlas configuration file."`
	HasDevURL        bool          `optional:"" help:"Whether the environment config has a dev_url attribute." default:"false"`
	SchemaScope      bool          `optional:"" help:"Limit the scope of the work done by Atlas (inspection, diffing, etc.) to one schema."`
	DirName          string        `optional:"" help:"Name of target migration directory in Atlas Cloud."`
	Replace          bool          `optional:"" help:"Replace existing Atlas CI workflow."`
	SetupSchemaApply *bool         `name:"schema-apply" help:"Whether to setup the 'schema apply' action."`
	driver           string        `hidden:"" help:"Driver of the migration directory (mysql,postgresql,mariadb,sqlite,sqlserver,clickhouse)."`
	flow             flowType      `hidden:"" help:"Workflow to initialize (versioned, declarative)."`
	stdin            io.ReadCloser `hidden:""`
	cloudURL         string        `hidden:""`
	cloudRepo        string        `hidden:""`
	env              gen.Env       `hidden:""`
}

func (i *InitActionCmd) Help() string {
	return `Examples:
	gh atlas init-action
	gh atlas init-action --token=$ATLAS_CLOUD_TOKEN
	gh atlas init-action --token=$ATLAS_CLOUD_TOKEN --dir-name="migrations" "dir/migrations"`
}

const (
	commitMsg = ".github/workflows: add atlas ci workflow"
	prBody    = "PR created by the `gh atlas init-action` command.\n for more information visit https://github.com/ariga/gh-atlas."
)

// Run the init-action command.
func (i *InitActionCmd) Run(ctx context.Context, client *githubClient, current repository.Repository) error {
	var (
		err        error
		randSuffix = randSeq(6)
		branchName = "atlas-ci-" + randSuffix
		secretName = "ATLAS_CLOUD_TOKEN_" + randSuffix
	)
	// validate params set by flags
	if err := i.validateParams(); err != nil {
		return err
	}
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
	if err = i.setToken(); err != nil {
		return err
	}
	cloud := cloudapi.New(i.cloudURL, i.Token)
	if err = cloud.ValidateToken(ctx); err != nil {
		return errors.New("the given Atlas token is invalid, please generate a new one and try again")
	}
	// inherit in case config is set by flags
	i.env.Path = i.ConfigPath
	i.env.Name = i.ConfigEnv
	if err = i.setParams(ctx, repo, cloud); err != nil {
		return err
	}
	if err = repo.SetSecret(ctx, secretName, i.Token); err != nil {
		return err
	}
	if err = repo.CheckoutNewBranch(ctx, branchName); err != nil {
		return err
	}
	cfg := &gen.Config{
		Flow:          string(i.flow),
		From:          i.From,
		To:            i.To,
		Path:          i.DirPath,
		DirName:       i.DirName,
		Driver:        i.driver,
		SecretName:    secretName,
		DefaultBranch: repo.defaultBranch,
		Env:           i.env,
		CreateDevURL:  !i.HasDevURL,
		SchemaScope:   i.SchemaScope,
		CloudRepo:     i.cloudRepo,
	}
	if i.flow == "declarative" {
		cfg.SetupSchemaApply = *i.SetupSchemaApply
	}
	if err = repo.AddAtlasYAML(ctx, cfg, branchName, commitMsg, i.Replace); err != nil {
		return err
	}
	link, err := repo.CreatePR(ctx, commitMsg, prBody, branchName)
	if err != nil {
		return err
	}
	fmt.Printf("%s %s %s\n",
		promptui.IconGood,
		promptui.Styler(promptui.FGFaint)("Created PR:"),
		link,
	)
	if err = i.openURL(link); err != nil {
		fmt.Printf("Failed to open %s in browser: %v\n", link, err)
	}
	return nil
}

var letters = []rune("0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
