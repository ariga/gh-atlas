package github

import (
	"context"
	"os"
	"time"

	"github.com/cli/go-gh"
	"github.com/cli/go-gh/pkg/auth"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v49/github"
)

const tempPath = "gh-atlas-temp"

type GitHubRepository struct {
	owner         string
	name          string
	repo          *git.Repository
	ghClient      *github.Client
	auth          *http.BasicAuth
	defaultBranch string
}

func NewGitHubRepository() (*GitHubRepository, error) {
	currRepo, err := gh.CurrentRepository()
	if err != nil {
		return nil, err
	}
	httpClient, err := gh.HTTPClient(nil)
	if err != nil {
		return nil, err
	}
	ghClient := github.NewClient(httpClient)
	repoData, _, err := ghClient.Repositories.Get(context.Background(), currRepo.Owner(), currRepo.Name())
	if err != nil {
		return nil, err
	}
	repo, err := git.PlainOpen(".")
	if err != nil {
		return nil, err
	}
	host, _ := auth.DefaultHost()
	token, _ := auth.TokenForHost(host)
	return &GitHubRepository{
		owner:    currRepo.Owner(),
		name:     currRepo.Name(),
		repo:     repo,
		ghClient: ghClient,
		auth: &http.BasicAuth{
			Username: "x-access-token",
			Password: token,
		},
		defaultBranch: repoData.GetDefaultBranch(),
	}, nil
}

// SetAtlasToken in repo secrets.
func (g *GitHubRepository) SetAtlasToken(token string) error {
	if token == "" {
		return nil
	}
	// TODO Implement logic to set the token in the repo secrets
	return nil
}

// CheckoutNewBranch creates a new branch in the repository.
func (g *GitHubRepository) CheckoutNewBranch(branchName string) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: branchRef,
	})
	return err
}

// AddAtlasYaml atlas.yaml file to staging area.
func (g *GitHubRepository) AddAtlasYaml(dirPath string) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	// TODO - implement logic to create the atlas.yaml file
	_, err = w.Filesystem.Create("temp-file.txt")
	if err != nil {
		return err
	}
	// TODO add the file to .github/workflows/atlas.yaml
	_, err = w.Add(".")
	return err
}

// CommitChanges commits the changes to the repository.
func (g *GitHubRepository) CommitChanges(commitMsg string) error {
	w, err := g.repo.Worktree()
	if err != nil {
		return err
	}
	user, _, err := g.ghClient.Users.Get(context.Background(), "")
	if err != nil {
		return err
	}
	_, err = w.Commit(commitMsg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  user.GetName(),
			Email: user.GetEmail(),
			When:  time.Now(),
		},
	})
	return err
}

// PushChanges pushes the changes to the remote repository.
func (g *GitHubRepository) PushChanges(branchName string) error {
	branchRef := plumbing.NewBranchReferenceName(branchName)
	err := g.repo.Push(&git.PushOptions{
		RemoteName: "origin",
		RefSpecs: []config.RefSpec{
			config.RefSpec(branchRef + ":" + branchRef),
		},
		Auth:     g.auth,
		Progress: os.Stdout,
	})
	return err
}

// CreatePR creates a pull request for the branch.
func (g *GitHubRepository) CreatePR(title string, branchName string) error {
	pr := &github.NewPullRequest{
		Title: &title,
		Head:  &branchName,
		Base:  &g.defaultBranch,
	}
	_, _, err := g.ghClient.PullRequests.Create(context.Background(), g.owner, g.name, pr)
	return err
}
