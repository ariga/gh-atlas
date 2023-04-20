package main

import (
	"context"
	"fmt"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v49/github"
)

type Repository struct {
	ctx           context.Context
	owner         string
	name          string
	defaultBranch string
	client        *github.Client
	auth          *http.BasicAuth
}

func NewRepository() (*Repository, error) {
	currRepo, err := gh.CurrentRepository()
	if err != nil {
		return nil, err
	}
	httpClient, err := gh.HTTPClient(nil)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(httpClient)
	ctx := context.Background()
	repoData, _, err := ghClient.Repositories.Get(ctx, currRepo.Owner(), currRepo.Name())
	if err != nil {
		return nil, err
	}
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	return &Repository{
		ctx:           ctx,
		owner:         currRepo.Owner(),
		name:          currRepo.Name(),
		defaultBranch: repoData.GetDefaultBranch(),
		client:        ghClient,
		auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: token,
		},
	}, nil
}

// CheckoutNewBranch creates a new branch on top of the default branch.
func (g *Repository) CheckoutNewBranch(branchName string) (*github.Reference, error) {
	defaultBranch, _, err := g.client.Git.GetRef(g.ctx, g.owner, g.name, "refs/heads/"+g.defaultBranch)
	if err != nil {
		return nil, err
	}
	newBranch := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: defaultBranch.Object.SHA,
		},
	}
	ref, _, err := g.client.Git.CreateRef(g.ctx, g.owner, g.name, newBranch)
	if err != nil {
		return nil, err
	}
	return ref, nil
}

// AddAtlasYaml atlas.yaml file to staging area.
func (g *Repository) AddAtlasYaml(dirPath, branchName string) error {
	// TODO implement this function
	newFile := &github.RepositoryContentFileOptions{
		Message: github.String("hello.txt"),
		Content: []byte(""),
		Branch:  github.String(branchName),
	}
	_, _, err := g.client.Repositories.CreateFile(g.ctx, g.owner, g.name, "./hello.txt", newFile)
	return err
}

// CommitChanges commits changes to the branch.
func (g *Repository) CommitChanges(branch *github.Reference, commitMsg string) error {
	latestCommit, _, err := g.client.Git.GetCommit(g.ctx, g.owner, g.name, branch.GetObject().GetSHA())
	if err != nil {
		return err
	}
	commit := &github.Commit{
		Message: github.String(commitMsg),
		Tree:    &github.Tree{SHA: latestCommit.GetTree().SHA},
		Parents: []*github.Commit{{
			SHA: branch.GetObject().SHA,
		}},
	}
	_, _, err = g.client.Git.CreateCommit(g.ctx, g.owner, g.name, commit)
	return err
}

// CreatePR creates a pull request for the branch.
func (g *Repository) CreatePR(title string, branchName string) error {
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &g.defaultBranch,
	}
	pr, _, err := g.client.PullRequests.Create(g.ctx, g.owner, g.name, newPR)
	if err != nil {
		return err
	}
	fmt.Println("created pull request:", pr.GetHTMLURL())
	return nil
}
