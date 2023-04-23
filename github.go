package main

import (
	"context"

	"ariga.io/gh-atlas/gen"
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

// AddAtlasYAML create commit with atlas ci yaml file on the branch.
func (r *Repository) AddAtlasYAML(dirPath, driver, branchName, commitMsg string) error {
	content, err := gen.Generate(&gen.Config{
		Path:          dirPath,
		DefaultBranch: r.defaultBranch,
		Driver:        driver,
	})
	if err != nil {
		return err
	}
	newFile := &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: content,
		Branch:  github.String(branchName),
	}
	_, _, err = r.client.Repositories.CreateFile(r.ctx, r.owner, r.name, ".github/workflows/ci-atlas.yaml", newFile)
	return err
}

// CreatePR creates a pull request for the branch and returns the link to the PR.
func (r *Repository) CreatePR(title string, branchName string) (string, error) {
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &r.defaultBranch,
	}
	pr, _, err := r.client.PullRequests.Create(r.ctx, r.owner, r.name, newPR)
	if err != nil {
		return "", err
	}
	return pr.GetHTMLURL(), nil
}
