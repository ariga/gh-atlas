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

	"ariga.io/gh-atlas/cloudapi"
	"ariga.io/gh-atlas/gen"
	"github.com/cli/go-gh/pkg/repository"
	"github.com/google/go-github/v49/github"
	"github.com/stretchr/testify/require"
)

// mockService is a mock implementation of necessary GitHub API methods.
type mockService struct {
	getContentError error
	hasHclFile      bool
	hclFileContent  string
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
func (m *mockService) GetContents(ctx context.Context, owner string, repo string, path string, opts *github.RepositoryContentGetOptions) (*github.RepositoryContent, []*github.RepositoryContent, *github.Response, error) {
	if path == "atlas.hcl" {
		return &github.RepositoryContent{Content: &m.hclFileContent}, nil, nil, nil
	}
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
	if m.hasHclFile {
		tree.Entries = append(tree.Entries, &github.TreeEntry{
			Path: github.String("atlas.hcl"),
			Type: github.String("blob"),
		})
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

func createGHClient(repoSvc repositoriesService, gitSvc gitService) *githubClient {
	return &githubClient{
		Git:          gitSvc,
		Repositories: repoSvc,
		Actions:      &mockService{},
		PullRequests: &mockService{},
	}
}

func TestRunInitActionCmd(t *testing.T) {
	repo, err := repository.Parse("owner/repo")
	reposByToken := map[string][]cloudapi.Repo{
		"one-repo": {
			{
				URL:    "atlas://name",
				Slug:   "name",
				Title:  "declarative_tests (Migration Directory)",
				Type:   "MIGRATION_DIRECTORY",
				Driver: "MYSQL",
			},
		},
		"multi-repos": {
			{
				URL:    "atlas://slug1",
				Slug:   "slug1",
				Type:   "SCHEMA",
				Title:  "declarative_tests (Schema)",
				Driver: "POSTGRESQL",
			},
			{
				URL:    "atlas://slug2",
				Slug:   "slug2",
				Type:   "MIGRATION_DIRECTORY",
				Title:  "darkhourse-initial (Migration Directory)",
				Driver: "MYSQL",
			},
			{
				URL:    "atlas://slug3",
				Slug:   "slug3",
				Type:   "SCHEMA",
				Title:  "pg (Schema)",
				Driver: "",
			},
		},
		"empty": {},
	}
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
					Repos []cloudapi.Repo `json:"repos"`
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
		if strings.Contains(input.Query, "repos") {
			token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			require.True(t, ok)
			payload.Data.Repos, ok = reposByToken[token]
			require.True(t, ok)
		}
		body, err := json.Marshal(payload)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}))
	require.NoError(t, err)
	defaultClient := createGHClient(&mockService{getContentError: &github.ErrorResponse{Message: "Not Found"}}, &mockService{})
	var tests = []struct {
		name     string
		client   *githubClient
		cmd      *InitActionCmd // initial command to run
		prompt   string         // user interaction with the terminal
		expected *InitActionCmd // expected command after user interaction
		wantErr  bool           // whether the command should return an error
	}{
		/*
			+---------------------+
			|    versioned flow   |
			+---------------------+
		*/
		{
			name: "all arg and flags supplied",
			cmd: &InitActionCmd{
				flow:        "versioned",
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			expected: &InitActionCmd{
				flow:        "versioned",
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "no dir path and driver supplied, choose from repo",
			cmd: &InitActionCmd{
				Token:       "one-repo",
				DirName:     "name",
				SchemaScope: true,
			},
			prompt: "\n\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "no dir path supplied, choose manual dir path",
			cmd: &InitActionCmd{
				Token:       "one-repo",
				DirName:     "name",
				SchemaScope: true,
			},
			// arrow key down, enter, `dir/migrations`, enter
			prompt: "\x1b[B\n`dir/migrations`\n\n",
			expected: &InitActionCmd{
				DirPath:     "`dir/migrations`",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "no dir name supplied use cloud dir name",
			cmd: &InitActionCmd{
				DirPath:     "migrations",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			prompt: "\na\n\n\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL", // this is the driver of the cloud repo
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "no dirs in organization",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				driver:  "MYSQL",
				Token:   "empty",
			},
			wantErr: true,
		},
		{
			name: "dir name provided but doesn't exist in cloud",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				driver:  "MYSQL",
				DirName: "invalid",
				Token:   "one-repo",
			},
			wantErr: true,
		},
		{
			name: "single dir in organization",
			cmd: &InitActionCmd{
				DirPath:     "migrations",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "multiple dirs in organization",
			cmd: &InitActionCmd{
				DirPath:     "migrations",
				driver:      "MYSQL",
				Token:       "multi-repos",
				SchemaScope: true,
			},
			// use arrow key down and then enter
			prompt: "\x1b[B\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "slug2",
				driver:      "MYSQL",
				Token:       "multi-repos",
				SchemaScope: true,
			},
		},
		{
			name: "no token flag supplied",
			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				SchemaScope: true,
			},
			prompt: "one-repo\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "empty token prompt",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				driver:  "MYSQL",
			},
			prompt:  " \n",
			wantErr: true,
		},
		{
			name: "invalid token prompt",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				driver:  "MYSQL",
			},
			prompt:  "invalid token\n",
			wantErr: true,
		},
		{
			name:   "ci file exists",
			client: createGHClient(&mockService{}, &mockService{}),
			prompt: "\n\n",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				driver:  "MYSQL",
				Token:   "one-repo",
			},
			wantErr: true,
		},
		{
			name:   "replace existing ci file",
			client: createGHClient(&mockService{}, &mockService{}),
			prompt: "\n\n",
			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				Replace:     true,
				SchemaScope: true,
			},
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				Replace:     true,
				SchemaScope: true,
			},
		},
		{
			name: "repo has atlas.hcl file, dont use it",
			client: createGHClient(
				&mockService{
					getContentError: &github.ErrorResponse{Message: "Not Found"},
					// language=HCL
					hclFileContent: `env "local" {
		                                  dev = "postgres://localhost:5432/dev"
		                            }`},
				&mockService{hasHclFile: true}),

			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			// arrow key down and then enter
			prompt: "\x1b[B\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "repo has atlas.hcl file, use it",
			client: createGHClient(
				&mockService{
					getContentError: &github.ErrorResponse{Message: "Not Found"},
					// language=HCL
					hclFileContent: `env "local" {
		                                  dev = "postgres://localhost:5432/dev"
		                            }`},
				&mockService{hasHclFile: true}),

			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			prompt: "\n\n\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
				env: gen.Env{
					Name:      "local",
					Path:      "atlas.hcl",
					HasDevURL: true,
				},
			},
		},
		{
			name: "repo has atlas.hcl without dev-url in env block",
			client: createGHClient(
				&mockService{
					getContentError: &github.ErrorResponse{Message: "Not Found"},
					// language=HCL
					hclFileContent: `env "local" {
		                                  url = "postgres://localhost:5432/dev"
		                            }`},
				&mockService{hasHclFile: true}),

			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			// arrow key down and then enter
			prompt: "\x1b[B\n\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
				env:         gen.Env{},
			},
		},
		{
			name: "repo has atlas.hcl without dev-url in env block, use it",
			client: createGHClient(
				&mockService{
					getContentError: &github.ErrorResponse{Message: "Not Found"},
					// language=HCL
					hclFileContent: `env "local" {
		                                  url = "postgres://localhost:5432/dev"
		                            }`},
				&mockService{hasHclFile: true}),

			cmd: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
			// arrow key down and then enter
			prompt: "\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				ConfigPath:  "atlas.hcl",
				ConfigEnv:   "local",
				SchemaScope: true,
				env: gen.Env{
					Name:   "local",
					Path:   "atlas.hcl",
					HasURL: true,
				},
			},
		},
		{
			name: "select single schema scope",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				driver:  "MYSQL",
				Token:   "one-repo",
			},
			prompt: "\n\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: true,
			},
		},
		{
			name: "select multiple schemas scope",
			cmd: &InitActionCmd{
				DirPath: "migrations",
				DirName: "name",
				driver:  "MYSQL",
				Token:   "one-repo",
			},
			// arrow key down, enter
			prompt: "\x1b[B\n",
			expected: &InitActionCmd{
				DirPath:     "migrations",
				DirName:     "name",
				driver:      "MYSQL",
				Token:       "one-repo",
				SchemaScope: false,
			},
		},
		/*
			+------------------------+
			|    declarative flow    |
			+------------------------+
		*/
		{
			name: "no from/to supplied",
			client: createGHClient(
				&mockService{
					hclFileContent: `env "local" {
		                                  dev = "mysql://localhost:3306/dev"
		                            }`},
				&mockService{hasHclFile: true}),
			cmd: &InitActionCmd{
				Token:       "multi-repos",
				SchemaScope: true,
				Replace:     true,
			},
			prompt: "\n\natlas://init\nfile://schema.hcl\n\n",
			expected: &InitActionCmd{
				From:             "atlas://init",
				To:               "file://schema.hcl",
				Token:            "multi-repos",
				driver:           "POSTGRESQL",
				SchemaScope:      true,
				cloudRepo:        "slug1",
				Replace:          true,
				SetupSchemaApply: ptr(true),
				env: gen.Env{
					Name:      "local",
					Path:      "atlas.hcl",
					HasDevURL: true,
				},
			},
		},
		{
			name: "all args supplied",
			cmd: &InitActionCmd{
				From:             "schema.hcl",
				To:               "atlas://slug1",
				driver:           "MYSQL",
				Token:            "multi-repos",
				SchemaScope:      true,
				SetupSchemaApply: ptr(true),
			},
			expected: &InitActionCmd{
				From:             "schema.hcl",
				To:               "atlas://slug1",
				driver:           "POSTGRESQL",
				Token:            "multi-repos",
				SchemaScope:      true,
				SetupSchemaApply: ptr(true),
			},
		},
		{
			name: "empty driver, select postgresql",
			cmd: &InitActionCmd{
				Token:            "multi-repos",
				SchemaScope:      true,
				SetupSchemaApply: ptr(true),
			},
			prompt: "\x1b[B\x1b[B\x1b[B\natlas://d\nfile://schema.hcl\n\x1b[B\n",
			expected: &InitActionCmd{
				Token:            "multi-repos",
				To:               "file://schema.hcl",
				driver:           "POSTGRESQL",
				From:             "atlas://d",
				cloudRepo:        "slug3",
				SchemaScope:      true,
				SetupSchemaApply: ptr(true),
			},
		},
		{
			name: "not setting 'schema apply' action by flag, select no",
			cmd: &InitActionCmd{
				Token:            "multi-repos",
				SchemaScope:      true,
				SetupSchemaApply: nil,
			},
			prompt: "\x1b[B\x1b[B\x1b[B\natlas://d\nfile://schema.hcl\n\x1b[B\n\x1b[B\n",
			expected: &InitActionCmd{
				Token:            "multi-repos",
				To:               "file://schema.hcl",
				driver:           "POSTGRESQL",
				From:             "atlas://d",
				cloudRepo:        "slug3",
				SchemaScope:      true,
				SetupSchemaApply: ptr(false),
			},
		},
		{
			name: "not setting 'schema apply' action by flag, select yes",
			cmd: &InitActionCmd{
				Token:            "multi-repos",
				SchemaScope:      true,
				SetupSchemaApply: nil,
			},
			prompt: "\x1b[B\x1b[B\x1b[B\natlas://d\nfile://schema.hcl\n\n\x1b[B\n",
			expected: &InitActionCmd{
				Token:            "multi-repos",
				To:               "file://schema.hcl",
				driver:           "POSTGRESQL",
				From:             "atlas://d",
				cloudRepo:        "slug3",
				SchemaScope:      true,
				SetupSchemaApply: ptr(true),
			},
		},
	}
	{
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				// set up the terminal interaction
				r, w, err := os.Pipe()
				require.NoError(t, err)
				_, err = w.WriteString(tt.prompt)
				require.NoError(t, err)
				err = w.Close()
				require.NoError(t, err)
				tt.cmd.stdin = &stdinBuffer{r}
				tt.cmd.cloudURL = srv.URL

				if tt.client == nil {
					tt.client = defaultClient
				}
				err = tt.cmd.Run(context.Background(), tt.client, repo)
				if tt.wantErr {
					require.Error(t, err)
					return
				}
				require.NoError(t, err)
				requireCommandsEqual(t, tt.expected, tt.cmd)
			})
		}
	}
}

func requireCommandsEqual(t *testing.T, a, b *InitActionCmd) {
	require.Equal(t, a.DirPath, b.DirPath, "DirPath mismatch")
	require.Equal(t, a.DirName, b.DirName, "DirName mismatch")
	require.Equal(t, a.driver, b.driver, "Driver mismatch")
	require.Equal(t, a.Token, b.Token, "Token mismatch")
	require.Equal(t, a.env, b.env, "env mismatch")
	require.Equal(t, a.Replace, b.Replace, "Replace mismatch")
	require.Equal(t, a.SchemaScope, b.SchemaScope, "SchemaScope mismatch")
	require.Equal(t, a.From, b.From, "From mismatch")
	require.Equal(t, a.To, b.To, "To mismatch")
	require.Equal(t, a.cloudRepo, b.cloudRepo, "cloudRepo mismatch")
	require.Equal(t, a.SetupSchemaApply, b.SetupSchemaApply, "SetupSchemaApply mismatch")
}
