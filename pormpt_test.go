package main

import (
	"context"
	"errors"
	"os"
	"testing"

	"ariga.io/gh-atlas/cloudapi"
	"ariga.io/gh-atlas/gen"
	"github.com/stretchr/testify/require"
)

func TestRunInitActionCmd_setConfigPath(t *testing.T) {
	var tests = []struct {
		name     string
		configs  []string       // input to the setConfigPath function
		prompt   string         // user interaction with the terminal
		expected *InitActionCmd // expected command after user interaction
		re       RepoExplorer
	}{
		{
			name:     "no config files",
			expected: &InitActionCmd{},
		},
		{
			name: "1 config file, dont use it",
			re: &mockRepoExplorer{
				content: `env "local" {}`,
			},
			configs: []string{"atlas.hcl"},
			// arrow key down and then enter
			prompt:   "\x1b[B\n\n",
			expected: &InitActionCmd{},
		},
		{
			name: "1 config file, use it",
			re: &mockRepoExplorer{
				content: `env "local" {
		                             dev = "postgres://localhost:5432/dev"
								}`,
			},
			configs: []string{"atlas.hcl"},
			prompt:  "\n",
			expected: &InitActionCmd{
				env: gen.Env{
					Name:      "local",
					Path:      "atlas.hcl",
					HasDevURL: true,
				},
			},
		},
		{
			name: "1 config file, witout dev url",
			re: &mockRepoExplorer{
				content: `env "local" {}`,
			},
			configs: []string{"atlas.hcl"},
			prompt:  "\n",
			expected: &InitActionCmd{
				env: gen.Env{
					Name: "local",
					Path: "atlas.hcl",
				},
			},
		},
		{
			name: "2 config files, use second",
			re: &mockRepoExplorer{
				content: `env "local" {
		                             dev = "postgres://localhost:5432/dev"
								}`,
			},
			configs: []string{"atlas.hcl", "atlas2.hcl"},
			// arrow key down, enter, enter
			prompt: "\x1b[B\n\n",
			expected: &InitActionCmd{
				env: gen.Env{
					Name:      "local",
					Path:      "atlas2.hcl",
					HasDevURL: true,
				},
			},
		},
		{
			name: " 2 config files, multiple envs, select first file and second env",
			re: &mockRepoExplorer{
				content: `
		env "local" {
		  	dev = "postgres://localhost:5432/dev"
		}
		env "prod" {
		  	dev = "postgres://localhost:5432/dev"
		}`,
			},
			configs: []string{"atlas.hcl", "atlas2.hcl"},
			// enter, arrow key down, enter
			prompt: "\n\x1b[B\n",
			expected: &InitActionCmd{
				env: gen.Env{
					Name:      "prod",
					HasDevURL: true,
					HasURL:    false,
					Path:      "atlas.hcl",
				},
			},
		},

		{
			name: " 2 config files, has unnamed env",
			re: &mockRepoExplorer{
				content: `
				env {
					name = atlas.env
					url = "postgres://localhost:5432/prod"
				  	dev = "postgres://localhost:5432/dev"
					schema {
						src = "file://./schema.sql"
						repo {
							name = "schema"
						}
					}
				}
		`,
			},
			configs: []string{"atlas.hcl", "atlas2.hcl"},
			// enter, arrow key down, enter
			prompt: "\nk8s\n",
			expected: &InitActionCmd{
				env: gen.Env{
					Name:         "k8s",
					Path:         "atlas.hcl",
					HasDevURL:    true,
					HasURL:       true,
					HasSchemaSrc: true,
					HasRepoName:  true,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// set up the terminal interaction
			r, w, err := os.Pipe()
			require.NoError(t, err)
			_, err = w.WriteString(tt.prompt)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)

			// run the command
			cmd := &InitActionCmd{
				stdin: &stdinBuffer{r},
			}
			err = cmd.setAtlasConfig(context.Background(), tt.configs, tt.re)
			require.NoError(t, err)
			requireCommandsEqual(t, tt.expected, cmd)
		})
	}
}

func TestRunInitActionCmd_selectRepo(t *testing.T) {
	tests := []struct {
		name         string
		repos        []cloudapi.Repo
		prompt       string
		expectedRepo *cloudapi.Repo
		expectedErr  error
	}{
		{
			name: "Single repo",
			repos: []cloudapi.Repo{
				{Title: "Repo1", URL: "url1", Type: cloudapi.SchemaType, Driver: "MYSQL"},
			},
			expectedRepo: &cloudapi.Repo{Title: "Repo1", URL: "url1", Type: cloudapi.SchemaType, Driver: "MYSQL"},
		},
		{
			name: "Multiple repos, select first",
			repos: []cloudapi.Repo{
				{Title: "Repo1", URL: "url1", Type: cloudapi.SchemaType, Driver: "MYSQL"},
				{Title: "Repo2", URL: "url2", Type: cloudapi.DirectoryType, Driver: "POSTGRESQL"},
			},
			prompt:       "\n",
			expectedRepo: &cloudapi.Repo{Title: "Repo1", URL: "url1", Type: cloudapi.SchemaType, Driver: "MYSQL"},
		},
		{
			name: "Multiple repos, select second",
			repos: []cloudapi.Repo{
				{Title: "Repo1", URL: "url1", Type: cloudapi.SchemaType, Driver: "MYSQL"},
				{Title: "Repo2", URL: "url2", Type: cloudapi.DirectoryType, Driver: "POSTGRESQL"},
			},
			prompt:       "\x1b[B\n",
			expectedRepo: &cloudapi.Repo{Title: "Repo2", URL: "url2", Type: cloudapi.DirectoryType, Driver: "POSTGRESQL"},
		},
		{
			name:        "No repos",
			repos:       []cloudapi.Repo{},
			expectedErr: errors.New("no repositories found"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err)
			_, err = w.WriteString(tt.prompt)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
			mockAPI := new(mockCloudAPI)
			mockAPI.repos = tt.repos
			cmd := &InitActionCmd{
				stdin: &stdinBuffer{r},
			}
			repo, err := cmd.selectAtlasRepo(context.Background(), mockAPI)
			if tt.expectedErr != nil {
				require.Error(t, err)
				require.Equal(t, tt.expectedErr.Error(), err.Error())
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.expectedRepo, repo)

		})
	}
}

func TestRunInitActionCmd_ChooseDirPath(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *InitActionCmd
		dirs     []string
		prompt   string
		expected string
	}{
		{
			name:     "choose first existing directory",
			cmd:      &InitActionCmd{},
			dirs:     []string{"migrations", "schema"},
			prompt:   "\n",
			expected: "migrations",
		},
		{
			name:     "choose second existing directory",
			cmd:      &InitActionCmd{},
			dirs:     []string{"migrations", "schema"},
			prompt:   "\x1b[B\n",
			expected: "schema",
		},
		{
			name:     "provide custom path",
			cmd:      &InitActionCmd{},
			dirs:     []string{"migrations", "schema"},
			prompt:   "\x1b[B\x1b[B\ncustom/path\n",
			expected: "custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, w, err := os.Pipe()
			require.NoError(t, err)
			_, err = w.WriteString(tt.prompt)
			require.NoError(t, err)
			err = w.Close()
			require.NoError(t, err)
			tt.cmd.stdin = &stdinBuffer{r}
			path, err := tt.cmd.chooseDirPath(tt.dirs)
			require.NoError(t, err)
			require.Equal(t, tt.expected, path)
		})
	}
}

type mockRepoExplorer struct {
	content  string
	cfgFiles []string
	dirs     []string
}

func (m *mockRepoExplorer) ReadContent(_ context.Context, _ string) (string, error) {
	return m.content, nil
}

func (m *mockRepoExplorer) MigrationDirectories(ctx context.Context) ([]string, error) {
	return m.dirs, nil
}

func (m *mockRepoExplorer) ConfigFiles(ctx context.Context) ([]string, error) {
	return m.cfgFiles, nil
}

type mockCloudAPI struct {
	repos []cloudapi.Repo
}

func (m *mockCloudAPI) Repos(ctx context.Context) ([]cloudapi.Repo, error) {
	return m.repos, nil
}

func (m *mockCloudAPI) ValidateToken(ctx context.Context) error {
	return nil
}
