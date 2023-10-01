package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"github.com/stretchr/testify/require"
)

// mockService is a mock implementation of necessary GitHub API methods.
type mockService struct {
	getContentError error
}

func (m *mockService) GetRef(context.Context, string, string, string) (*github.Reference, *github.Response, error) {
	ref := &github.Reference{
		Object: &github.GitObject{
			SHA: nil,
		},
	}
	return ref, nil, nil
}
func (m *mockService) CreateRef(context.Context, string, string, *github.Reference) (*github.Reference, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) Get(context.Context, string, string) (*github.Repository, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) GetContents(context.Context, string, string, string, *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	sha := "12345"
	return &github.RepositoryContent{SHA: &sha}, nil, nil, m.getContentError
}
func (m *mockService) CreateFile(context.Context, string, string, string, *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) GetRepoSecret(context.Context, string, string, string) (*github.Secret, *github.Response, error) {
	res := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusNotFound,
		},
	}
	return nil, res, nil
}
func (m *mockService) CreateOrUpdateRepoSecret(context.Context, string, string, *github.EncryptedSecret) (*github.Response, error) {
	res := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
	return res, nil
}
func (m *mockService) GetRepoPublicKey(context.Context, string, string) (*github.PublicKey, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) Create(context.Context, string, string, *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) GetTree(context.Context, string, string, string, bool) (*github.Tree, *github.Response, error) {
	tree := &github.Tree{
		Entries: []*github.TreeEntry{
			{
				Path: github.String("migrations/atlas.sum"),
				Type: github.String("blob"),
			},
		},
	}
	return tree, nil, nil
}

type stdinBuffer struct {
	io.Reader
}

func (b *stdinBuffer) Close() error {
	return nil
}

// read one byte at a time, default reader of promptui package reads 4096 bytes at a time
func (b *stdinBuffer) Read(dst []byte) (int, error) {
	buf := make([]byte, 1)
	n, err := b.Reader.Read(buf)
	if err != nil {
		return n, err
	}
	copy(dst, buf)
	return n, nil
}

func createGHClient(repoSvc repositoriesService) *githubClient {
	return &githubClient{
		Git:          &mockService{},
		Repositories: repoSvc,
		Actions:      &mockService{},
		PullRequests: &mockService{},
	}
}

func TestRunInitActionCmd(t *testing.T) {
	repo, err := repository.Parse("owner/repo")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			input struct {
				Query     string `json:"query,omitempty"`
				Variables struct {
					Token string `json:"token"`
				} `json:"variables"`
			}
			payload struct {
				Data struct {
					DirSlugs []string `json:"dirSlugs"`
				} `json:"data"`
			}
		)
		err = json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		if strings.Contains(input.Query, "validateToken") {
			if input.Variables.Token == "invalid token" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
		}
		if strings.Contains(input.Query, "dirSlugs") {
			payload.Data.DirSlugs = []string{"name"}
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}))
	require.NoError(t, err)
	var tests = []struct {
		name     string
		client   *githubClient
		cmd      *InitActionCmd // initial command to run
		prompt   string         // user interaction with the terminal
		expected *InitActionCmd // expected command after user interaction
		wantErr  bool           // whether the command should return an error
	}{
		{
			name:   "all arg and flags supplied",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
			},
			expected: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
			},
		},
		{
			name:   "no dir path and driver supplied",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				Token:   "token",
				DirName: "name",
			},
			prompt: "\n\n",
			expected: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
			},
		},
		{
			name:   "no dir name supplied use cloud dir name",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				Driver:  "mysql",
				Token:   "token",
			},
			prompt: "\n\n",
			expected: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
			},
		},
		{
			name:   "no dir name supplied dont use cloud",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				Driver:  "mysql",
				Token:   "token",
			},
			// use arrow key down and then enter
			prompt: "\x1b[B\n\n",
			expected: &InitActionCmd{
				DirPath: "migrations",
				Driver:  "mysql",
				Token:   "token",
			},
		},
		{
			name:   "no token flag supplied",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
			},
			prompt: "my token\n",
			expected: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "my token",
			},
		},
		{
			name:   "empty token prompt",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				Driver:  "mysql",
			},
			prompt:  " \n",
			wantErr: true,
		},
		{
			name:   "invalid token prompt",
			client: createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}),
			cmd: &InitActionCmd{
				DirPath: "migrations",
				Driver:  "mysql",
			},
			prompt:  "invalid token\n",
			wantErr: true,
		},
		{
			name:   "file exists",
			client: createGHClient(&mockService{}),
			prompt: "my token\n",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
			},
			wantErr: true,
		},
		{
			name:   "replace existing file",
			client: createGHClient(&mockService{}),
			prompt: "my token\n",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
				Replace: true,
			},
			expected: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				Driver:  "mysql",
				Token:   "token",
				Replace: true,
			},
		},
	}
	{
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				r, w, err := os.Pipe()
				require.NoError(t, err)
				_, err = w.WriteString(tt.prompt)
				require.NoError(t, err)
				err = w.Close()
				require.NoError(t, err)
				tt.cmd.stdin = &stdinBuffer{r}
				tt.cmd.cloudURL = srv.URL

				err = tt.cmd.Run(context.Background(), tt.client, repo)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				require.Equal(t, tt.expected.Token, tt.cmd.Token)
				require.Equal(t, tt.expected.Driver, tt.cmd.Driver)
				require.Equal(t, tt.expected.DirPath, tt.cmd.DirPath)
				require.Equal(t, tt.expected.DirName, tt.cmd.DirName)
				require.Equal(t, tt.expected.Replace, tt.cmd.Replace)
			})
		}
	}
}
