package main

import (
	"context"
	"fmt"

	"github.com/cli/go-gh"
	"github.com/google/go-github/v49/github"
)

type Repository struct {
	ctx           context.Context
	owner         string
	name          string
	defaultBranch string
	client        *github.Client
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
	return &Repository{
		ctx:           ctx,
		owner:         currRepo.Owner(),
		name:          currRepo.Name(),
		defaultBranch: repoData.GetDefaultBranch(),
		client:        ghClient,
	}, nil
}

// CheckoutNewBranch creates a new branch on top of the default branch.
func (g *Repository) CheckoutNewBranch(branchName string) error {
	defaultBranch, _, err := g.client.Git.GetRef(g.ctx, g.owner, g.name, "refs/heads/"+g.defaultBranch)
	if err != nil {
		return err
	}
	newBranch := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: defaultBranch.Object.SHA,
		},
	}
	_, _, err = g.client.Git.CreateRef(g.ctx, g.owner, g.name, newBranch)
	return err
}

// AddAtlasYaml create commit with atlas ci yaml file on the branch.
func (g *Repository) AddAtlasYaml(dirPath, branchName, commitMsg string) error {
	// TODO implement yaml file creation
	newFile := &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: []byte(""),
		Branch:  github.String(branchName),
	}
	resp, _, err := g.client.Repositories.CreateFile(g.ctx, g.owner, g.name, "./hello.txt", newFile)
	resp.Commit.Message = github.String("hello.txt")
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
