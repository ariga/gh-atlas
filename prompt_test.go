package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockPrompt struct{}

func (m *mockPrompt) choose(msg string, _ []string) (string, error) {
	switch msg {
	case "choose migration directory":
		return "set-dir", nil
	case "choose driver":
		return "set-driver", nil
	}
	return "", nil
}

func (m *mockPrompt) input(msg string) (string, error) {
	if msg == "enter Atlas Cloud token" {
		return "set-token", nil
	}
	return "", nil
}

func TestSetParams(t *testing.T) {
	var tests = []struct {
		name     string
		cmd      *InitCiCmd
		expected *InitCiCmd
	}{
		{
			name: "all arg and flags supplied",
			cmd: &InitCiCmd{
				DirPath: "my-path",
				Driver:  "my-driver",
				Token:   "my-token",
			},
			expected: &InitCiCmd{
				DirPath: "my-path",
				Driver:  "my-driver",
				Token:   "my-token",
			},
		},
		{
			name: "no arg or flags supplied",
			cmd:  &InitCiCmd{},
			expected: &InitCiCmd{
				DirPath: "set-dir",
				Driver:  "set-driver",
				Token:   "set-token",
			},
		},
		{
			name: "only token flag supplied",
			cmd: &InitCiCmd{
				Token: "my-token",
			},
			expected: &InitCiCmd{
				DirPath: "set-dir",
				Driver:  "set-driver",
				Token:   "my-token",
			},
		},
	}
	{
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := setParams(tt.cmd, nil, &mockPrompt{})
				require.NoError(t, err)
				require.Equal(t, tt.expected, tt.cmd)
			})
		}
	}
}
