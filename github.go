package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"ariga.io/gh-atlas/gen"
	"github.com/cli/go-gh"
	"github.com/google/go-github/v49/github"
	"golang.org/x/crypto/nacl/box"
)

type (
	// gitService handles communication with the git data related methods of the GitHub API.
	gitService interface {
		GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
		CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
	}
	// repositoriesService handles communication with the repository related methods of the GitHub API.
	repositoriesService interface {
		Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
		CreateFile(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error)
	}
	// actionsService handles communication with the actions related methods of the GitHub API.
	actionsService interface {
		GetRepoSecret(ctx context.Context, owner, repo, name string) (*github.Secret, *github.Response, error)
		CreateOrUpdateRepoSecret(ctx context.Context, owner, repo string, eSecret *github.EncryptedSecret) (*github.Response, error)
		GetRepoPublicKey(ctx context.Context, owner, repo string) (*github.PublicKey, *github.Response, error)
	}
	// pullRequestsService handles communication with the pull request related methods of the GitHub API.
	pullRequestsService interface {
		Create(ctx context.Context, owner, repo string, pr *github.NewPullRequest) (*github.PullRequest, *github.Response, error)
	}
	// githubClient is a wrapper around the GitHub API client.
	githubClient struct {
		Git          gitService
		Repositories repositoriesService
		Actions      actionsService
		PullRequests pullRequestsService
	}
)

type Repository struct {
	ctx           context.Context
	owner         string
	name          string
	defaultBranch string
	client        *githubClient
}

// NewRepository creates a new repository object.
func NewRepository(client *githubClient) (*Repository, error) {
	currRepo, err := gh.CurrentRepository()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	repoData, _, err := client.Repositories.Get(ctx, currRepo.Owner(), currRepo.Name())
	if err != nil {
		return nil, err
	}
	return &Repository{
		ctx:           ctx,
		owner:         currRepo.Owner(),
		name:          currRepo.Name(),
		defaultBranch: repoData.GetDefaultBranch(),
		client:        client,
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

// SetSecret sets Secret for the repository with the given name and value.
// if the secret already exists, it will not be updated.
func (r *Repository) SetSecret(name, value string) error {
	_, res, err := r.client.Actions.GetRepoSecret(r.ctx, r.owner, r.name, name)
	if err != nil && res.StatusCode != http.StatusNotFound {
		return err
	}
	if res.StatusCode == http.StatusOK {
		fmt.Printf("secret %q already exists\n", name)
		return nil
	}
	key, _, err := r.client.Actions.GetRepoPublicKey(r.ctx, r.owner, r.name)
	if err != nil {
		return err
	}
	decodedPK, err := base64.StdEncoding.DecodeString(key.GetKey())
	if err != nil {
		return err
	}
	// Convert the decodedPK to a usable format of *[32]byte which is required by the box.SealAnonymous function.
	var publicKey [32]byte
	copy(publicKey[:], decodedPK)
	encrypted, err := box.SealAnonymous(nil, []byte(value), &publicKey, nil)
	if err != nil {
		return errors.New("failed to encrypt secret value")
	}
	secret := &github.EncryptedSecret{
		Name:           name,
		KeyID:          key.GetKeyID(),
		EncryptedValue: base64.StdEncoding.EncodeToString(encrypted),
	}
	res, err = r.client.Actions.CreateOrUpdateRepoSecret(r.ctx, r.owner, r.name, secret)
	if res.StatusCode == http.StatusForbidden {
		return errors.New("forbidden: make sure you have access to set secrets for this repository")
	}
	return err
}

// AddAtlasYAML create commit with atlas ci yaml file on the branch.
func (r *Repository) AddAtlasYAML(cfg *gen.Config, branchName, commitMsg string) error {
	content, err := gen.Generate(cfg)
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
