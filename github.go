package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"ariga.io/gh-atlas/gen"
	"github.com/cli/go-gh"
	"github.com/google/go-github/v49/github"
	"golang.org/x/crypto/nacl/box"
)

// GitHubRepository is the interface for interacting with a GitHub repository.
type GitHubRepository interface {
	CheckoutNewBranch(branchName string) error
	SetSecret(name, value string) error
	AddAtlasYAML(cfg *gen.Config, branchName, commitMsg string) error
	CreatePR(title string, branchName string) (string, error)
	MigrationDirectories() ([]string, error)
}

// Repository is the implementation of the GitHubRepository interface.
type Repository struct {
	ctx           context.Context
	owner         string
	name          string
	defaultBranch string
	client        *github.Client
}

// NewRepository returns a new GitHubRepository.
func NewRepository(cmdCtx *Context) (GitHubRepository, error) {
	if cmdCtx.Testing {
		return &MockRepo{}, nil
	}
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
	cfg.DefaultBranch = r.defaultBranch
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

// MigrationDirectories returns a list of paths to directories containing migration files.
func (r *Repository) MigrationDirectories() ([]string, error) {
	t, _, err := r.client.Git.GetTree(r.ctx, r.owner, r.name, r.defaultBranch, true)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range t.Entries {
		if e.GetType() == "blob" && strings.HasSuffix(e.GetPath(), "atlas.sum") {
			paths = append(paths, path.Dir(e.GetPath()))
		}
	}
	return paths, nil
}

// MockRepo is a mock implementation of the Repo interface used for testing.
type MockRepo struct{}

func (r *MockRepo) CheckoutNewBranch(string) error {
	return nil
}
func (r *MockRepo) SetSecret(string, string) error {
	return nil
}
func (r *MockRepo) AddAtlasYAML(*gen.Config, string, string) error {
	return nil
}
func (r *MockRepo) CreatePR(string, string) (string, error) {
	return "", nil
}
func (r *MockRepo) MigrationDirectories() ([]string, error) {
	return []string{"dir1", "dir2"}, nil
}
