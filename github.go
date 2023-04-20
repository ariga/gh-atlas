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
func (r *Repository) CheckoutNewBranch(branchName string) error {
	defaultBranch, _, err := r.client.Git.GetRef(r.ctx, r.owner, r.name, "refs/heads/"+r.defaultBranch)
	if err != nil {
		return err
	}
	newBranch := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: defaultBranch.Object.SHA,
		},
	}
	_, _, err = r.client.Git.CreateRef(r.ctx, r.owner, r.name, newBranch)
	return err
}

// AddAtlasYaml create commit with atlas ci yaml file on the branch.
func (r *Repository) AddAtlasYaml(dirPath, branchName, commitMsg string) error {
	// TODO implement yaml file creation
	newFile := &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: []byte(""),
		Branch:  github.String(branchName),
	}
	_, _, err := r.client.Repositories.CreateFile(r.ctx, r.owner, r.name, "./hello.txt", newFile)
	return err
}

// CreatePR creates a pull request for the branch.
func (r *Repository) CreatePR(title string, branchName string) error {
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &r.defaultBranch,
	}
	pr, _, err := r.client.PullRequests.Create(r.ctx, r.owner, r.name, newPR)
	if err != nil {
		return err
	}
	fmt.Println("created pull request:", pr.GetHTMLURL())
	return nil
}
