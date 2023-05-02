package main

import (
	"fmt"
	"math/rand"

	"ariga.io/gh-atlas/gen"
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
	DirPath string `arg:"" type:"-path" help:"Path inside repository containing the migration files."`
	Driver  string `enum:"mysql,postgres,mariadb,sqlite" default:"mysql" help:"Driver of the migration directory."`
	Token   string `required:"" short:"t" help:"(Required) Atlas authentication token."`
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
func (i *InitCiCmd) Run() error {
	var (
		branchName = "atlas-ci-" + randSeq(6)
		secretName = "ATLAS_CLOUD_TOKEN"
	)
	repo, err := NewRepository()
	if err != nil {
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
