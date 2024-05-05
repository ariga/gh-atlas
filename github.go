package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"golang.org/x/crypto/nacl/box"

	"ariga.io/gh-atlas/gen"
)

type (
	// gitService handles communication with the git data related methods of the GitHub API.
	gitService interface {
		GetRef(ctx context.Context, owner string, repo string, ref string) (*github.Reference, *github.Response, error)
		CreateRef(ctx context.Context, owner string, repo string, ref *github.Reference) (*github.Reference, *github.Response, error)
		GetTree(ctx context.Context, owner string, repo string, sha string, recursive bool) (*github.Tree, *github.Response, error)
	}
	// repositoriesService handles communication with the repository related methods of the GitHub API.
	repositoriesService interface {
		Get(ctx context.Context, owner, repo string) (*github.Repository, *github.Response, error)
		GetContents(ctx context.Context, owner, repo, path string, opts *github.RepositoryContentGetOptions) (fileContent *github.RepositoryContent, directoryContent []*github.RepositoryContent, resp *github.Response, err error)
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
	owner         string
	name          string
	defaultBranch string
	client        *githubClient
}

// NewRepository creates a new repository object.
func NewRepository(client *githubClient, current repository.Repository, defaultBranch string) *Repository {
	return &Repository{
		owner:         current.Owner(),
		name:          current.Name(),
		defaultBranch: defaultBranch,
		client:        client,
	}
}

// CheckoutNewBranch creates a new branch on top of the default branch.
func (r *Repository) CheckoutNewBranch(ctx context.Context, branchName string) error {
	defaultBranch, _, err := r.client.Git.GetRef(ctx, r.owner, r.name, "refs/heads/"+r.defaultBranch)
	if err != nil {
		return err
	}
	newBranch := &github.Reference{
		Ref: github.String("refs/heads/" + branchName),
		Object: &github.GitObject{
			SHA: defaultBranch.Object.SHA,
		},
	}
	_, _, err = r.client.Git.CreateRef(ctx, r.owner, r.name, newBranch)
	return err
}

// SetSecret sets Secret for the repository with the given name and value.
// if the secret already exists, it will not be updated.
func (r *Repository) SetSecret(ctx context.Context, name, value string) error {
	_, res, err := r.client.Actions.GetRepoSecret(ctx, r.owner, r.name, name)
	if err != nil && res.StatusCode != http.StatusNotFound {
		return err
	}
	if res.StatusCode == http.StatusOK {
		return fmt.Errorf("secret %q already exists", name)
	}
	key, _, err := r.client.Actions.GetRepoPublicKey(ctx, r.owner, r.name)
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
	res, err = r.client.Actions.CreateOrUpdateRepoSecret(ctx, r.owner, r.name, secret)
	if res.StatusCode == http.StatusForbidden {
		return errors.New("forbidden: make sure you have access to set secrets for this repository")
	}
	return err
}

// AddAtlasYAML create commit with atlas ci yaml file on the branch.
func (r *Repository) AddAtlasYAML(ctx context.Context, cfg *gen.Config, branchName, commitMsg string, replace bool) error {
	var actionFilePath = ".github/workflows/ci-atlas.yaml"
	content, err := gen.Generate(cfg)
	if err != nil {
		return err
	}
	newFile := &github.RepositoryContentFileOptions{
		Message: github.String(commitMsg),
		Content: content,
		Branch:  github.String(branchName),
	}
	current, _, _, err := r.client.Repositories.GetContents(ctx, r.owner, r.name, actionFilePath, nil)
	switch e := err.(type) {
	case nil:
		if !replace {
			return errors.New("atlas ci yaml file already exists, use --replace to replace it")
		}
		newFile.SHA = current.SHA
	case *github.ErrorResponse:
		if e.Message != "Not Found" {
			return err
		}
	default:
		return err
	}
	_, _, err = r.client.Repositories.CreateFile(ctx, r.owner, r.name, actionFilePath, newFile)
	return err
}

// CreatePR creates a pull request for the branch and returns the link to the PR.
func (r *Repository) CreatePR(ctx context.Context, title string, body string, branchName string) (string, error) {
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Body:  &body,
		Base:  &r.defaultBranch,
	}
	pr, _, err := r.client.PullRequests.Create(ctx, r.owner, r.name, newPR)
	if err != nil {
		return "", err
	}
	return pr.GetHTMLURL(), nil
}

// MigrationDirectories returns a list of paths to directories containing migration files.
func (r *Repository) MigrationDirectories(ctx context.Context) ([]string, error) {
	t, _, err := r.client.Git.GetTree(ctx, r.owner, r.name, r.defaultBranch, true)
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

// ConfigFiles returns a list of paths to atlas.hcl files in the repository.
func (r *Repository) ConfigFiles(ctx context.Context) ([]string, error) {
	t, _, err := r.client.Git.GetTree(ctx, r.owner, r.name, r.defaultBranch, true)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range t.Entries {
		if e.GetType() == "blob" && strings.HasSuffix(e.GetPath(), "atlas.hcl") {
			paths = append(paths, e.GetPath())
		}
	}
	return paths, nil
}

// Implementation of the ContentReader interface.
func (r *Repository) ReadContent(ctx context.Context, path string) (string, error) {
	fileContents, _, _, err := r.client.Repositories.GetContents(ctx, r.owner, r.name, path, nil)
	if err != nil {
		return "", err
	}
	return fileContents.GetContent()
}
