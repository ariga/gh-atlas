package main

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetConfigPath(t *testing.T) {
	var tests = []struct {
		name     string
		configs  []string       // input to the setConfigPath function
		cr       ContentReader  // input to the setConfigPath function
		prompt   string         // user interaction with the terminal
		expected *InitActionCmd // expected command after user interaction
	}{
		{
			name:     "no config files",
			configs:  []string{},
			expected: &InitActionCmd{},
		},
		{
			name: "1 config file, dont use it",
			cr: &mockContentReader{
				content: `env "local" {}`,
			},
			configs: []string{"atlas.hcl"},
			// arrow key down and then enter
			prompt: "\x1b[B\n\n",
			expected: &InitActionCmd{
				ConfigPath: "",
			},
		},
		{
			name: "1 config file, use it",
			cr: &mockContentReader{
				content: `env "local" {
                             dev = "postgres://localhost:5432/dev"
						}`,
			},
			configs: []string{"atlas.hcl"},
			prompt:  "\n",
			expected: &InitActionCmd{
				ConfigPath: "atlas.hcl",
				ConfigEnv:  "local",
				HasDevURL:  true,
			},
		},
		{
			name: "1 config file, witout dev url",
			cr: &mockContentReader{
				content: `env "local" {}`,
			},
			configs: []string{"atlas.hcl"},
			prompt:  "\n",
			expected: &InitActionCmd{
				ConfigPath: "atlas.hcl",
				ConfigEnv:  "local",
			},
		},
		{
			name: "2 config files, use second",
			cr: &mockContentReader{
				content: `env "local" {
                             dev = "postgres://localhost:5432/dev"
						}`,
			},
			configs: []string{"atlas.hcl", "atlas2.hcl"},
			// arrow key down, enter, enter
			prompt: "\x1b[B\n\n",
			expected: &InitActionCmd{
				ConfigPath: "atlas2.hcl",
				ConfigEnv:  "local",
				HasDevURL:  true,
			},
		},
		{
			name: " 2 config files, multiple envs, select first file and second env",
			cr: &mockContentReader{
				content: `env "local" {
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
				ConfigPath: "atlas.hcl",
				ConfigEnv:  "prod",
				HasDevURL:  true,
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
			err = cmd.setConfigPath(context.Background(), tt.configs, tt.cr)
			require.NoError(t, err)
			requireCommandsEqual(t, tt.expected, cmd)
		})
	}
}

type mockContentReader struct {
	content string
}

func (m *mockContentReader) ReadContent(_ context.Context, _ string) (string, error) {
	return m.content, nil
}
