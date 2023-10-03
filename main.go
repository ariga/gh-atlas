package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"slices"

	"github.com/alecthomas/kong"
	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"github.com/pkg/browser"

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
	DirPath  string        `arg:"" optional:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver   string        `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory (mysql,postgres,mariadb,sqlite)."`
	Token    string        `short:"t" help:"Atlas authentication token."`
	Repo     string        `short:"R" help:"GitHub repository owner/name, defaults to the current repository."`
	DirName  string        `optional:"" help:"Name of target migration directory in Atlas Cloud."`
	Replace  bool          `optional:"" help:"Replace existing Atlas CI workflow."`
	stdin    io.ReadCloser `hidden:""`
	cloudURL string        `hidden:""`
}

func (i *InitActionCmd) Help() string {
	return `Examples:
	gh atlas init-action
	gh atlas init-action --token=$ATLAS_CLOUD_TOKEN
	gh atlas init-action --token=$ATLAS_CLOUD_TOKEN --dir-name="migrations" --driver="mysql" "dir/migrations"`
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
	cloud := cloudapi.New(i.cloudURL, i.Token)
	if err = cloud.ValidateToken(ctx); err != nil {
		return errors.New("the given atlas token is invalid, please generate a new one and try again")
	}
	dirNames, err := cloud.DirNames(ctx)
	if err != nil {
		return err
	}
	if len(dirNames) == 0 {
		return errors.New("no migration directories found in your organization")
	}
	if i.DirName != "" && !slices.Contains(dirNames, i.DirName) {
		return fmt.Errorf("no migration directory with name %q found in your organization", i.DirName)
	}
	// If the dir name was not provided by the user, set it interactively.
	if err := i.setDirName(dirNames); err != nil {
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
		DirName:       i.DirName,
		Driver:        i.Driver,
		SecretName:    secretName,
		DefaultBranch: repo.defaultBranch,
	}
	if err = repo.AddAtlasYAML(ctx, cfg, branchName, commitMsg, i.Replace); err != nil {
		return err
	}
	link, err := repo.CreatePR(ctx, commitMsg, prBody, branchName)
	if err != nil {
		return err
	}
	fmt.Println("Created PR:", link)
	if err = browser.OpenURL(link); err != nil {
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
