package main

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	pseudotty "github.com/creack/pty"
	"github.com/google/go-github/v49/github"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
)

// mockService is a mock implementation of necessary GitHub API methods.
type mockService struct{}

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
func (m *mockService) Get(context.Context, string, string) (*github.Repository, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) CreateFile(context.Context, string, string, string, *github.RepositoryContentFileOptions) (*github.RepositoryContentResponse, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) GetRepoSecret(context.Context, string, string, string) (*github.Secret, *github.Response, error) {
	res := &github.Response{
		Response: &http.Response{
			StatusCode: http.StatusOK,
		},
	}
	return nil, res, nil
}
func (m *mockService) CreateOrUpdateRepoSecret(context.Context, string, string, *github.EncryptedSecret) (*github.Response, error) {
	return nil, nil
}
func (m *mockService) GetRepoPublicKey(context.Context, string, string) (*github.PublicKey, *github.Response, error) {
	return nil, nil, nil
}
func (m *mockService) Create(context.Context, string, string, *github.NewPullRequest) (*github.PullRequest, *github.Response, error) {
	return nil, nil, nil
}

func TestRunInitCICmd(t *testing.T) {
	var tests = []struct {
		name     string
		cmd      *InitCiCmd            // initial command to run
		prompt   func(*expect.Console) // user interaction with the terminal
		expected *InitCiCmd            // expected command after user interaction
	}{
		{
			name: "all arg and flags supplied",
			cmd: &InitCiCmd{
				DirPath: "path",
				Driver:  "postgres",
				Token:   "token",
			},
			prompt: func(c *expect.Console) {
				_, err := c.ExpectEOF()
				require.NoError(t, err)
			},
			expected: &InitCiCmd{
				DirPath: "path",
				Driver:  "postgres",
				Token:   "token",
			},
		},
		{
			name: "no token flag supplied",
			cmd: &InitCiCmd{
				DirPath: "path",
				Driver:  "postgres",
			},
			prompt: func(c *expect.Console) {
				_, err := c.ExpectString("enter Atlas Cloud token")
				require.NoError(t, err)
				_, err = c.Send("token" + string(terminal.KeyEnter))
				require.NoError(t, err)
				_, err = c.ExpectEOF()
				require.NoError(t, err)
			},
			expected: &InitCiCmd{
				DirPath: "path",
				Driver:  "postgres",
				Token:   "token",
			},
		},
		{
			name: "no dir path and driver supplied",
			cmd: &InitCiCmd{
				Token: "token",
			},
			prompt: func(c *expect.Console) {
				_, err := c.ExpectString("choose migration directory")
				require.NoError(t, err)
				_, err = c.Send(string(terminal.KeyArrowDown) + string(terminal.KeyEnter))
				require.NoError(t, err)
				_, err = c.ExpectString("choose driver")
				require.NoError(t, err)
				_, err = c.Send(string(terminal.KeyArrowDown) + string(terminal.KeyEnter))
				require.NoError(t, err)
				_, err = c.ExpectEOF()
				require.NoError(t, err)
			},
			expected: &InitCiCmd{
				DirPath: "migrations",
				Driver:  "postgres",
				Token:   "token",
			},
		},
	}
	{
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				RunTest(t, tt.prompt, func() error {
					client := &githubClient{
						Git:          &mockService{},
						Repositories: &mockService{},
						Actions:      &mockService{},
						PullRequests: &mockService{},
					}
					return tt.cmd.Run(client)
				})
				require.Equal(t, tt.expected, tt.cmd)
			})
		}
	}
}

// RunTest runs a given test with expected I/O prompt.
func RunTest(t *testing.T, prompt func(*expect.Console), test func() error) {
	pty, tty, err := pseudotty.Open()
	require.NoError(t, err)
	// virtual terminal needed for way to interpret terminal / ANSI escape sequences
	// for more info see: https://github.com/go-survey/survey#testing
	term := vt10x.New(vt10x.WithWriter(tty))
	c, err := expect.NewConsole(expect.WithStdin(pty), expect.WithStdout(term), expect.WithCloser(pty, tty))
	require.NoError(t, err)
	defer c.Close()
	done := make(chan struct{})
	go func() {
		defer close(done)
		prompt(c)
	}()
	// replace stdin and stdout with the tty so that the user can interact with the console
	originalStdin := os.Stdin
	originalStdout := os.Stdout
	os.Stdin = c.Tty()
	os.Stdout = c.Tty()
	defer func() {
		os.Stdin = originalStdin
		os.Stdout = originalStdout
	}()
	err = test()
	require.NoError(t, err)
	err = c.Tty().Close()
	require.NoError(t, err)
	<-done
}
