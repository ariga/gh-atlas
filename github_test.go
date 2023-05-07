package main

import (
	"context"
	"testing"

	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"github.com/google/go-replayers/httpreplay"
	"github.com/stretchr/testify/require"
)

func TestGitHubMigrationDirectories(t *testing.T) {
	r, err := httpreplay.NewReplayer("./testdata/github_dirs.json")
	require.NoError(t, err)
	defer r.Close()
	client := github.NewClient(r.Client())
	ghClient := &githubClient{
		Git:          client.Git,
		Repositories: client.Repositories,
		Actions:      client.Actions,
		PullRequests: client.PullRequests,
	}
	currRepo, err := repository.Parse("rotemtam/atlas-demo")
	require.NoError(t, err)
	repo := NewRepository(context.Background(), ghClient, currRepo, "main")
	dirs, err := repo.MigrationDirectories()
	require.NoError(t, err)
	require.Len(t, dirs, 2)
}
