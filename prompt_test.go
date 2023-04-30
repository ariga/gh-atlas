package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type mockPrompt struct {
	chooseCalled int
	inputCalled  int
}

func (m *mockPrompt) choose(_ string, _ []string) (string, error) {
	m.chooseCalled++
	return "", nil
}

func (m *mockPrompt) input(_ string) (string, error) {
	m.inputCalled++
	return "", nil
}

func TestSetParams(t *testing.T) {
	t.Run("all arg and flags supplied", func(t *testing.T) {
		cmd := &InitCiCmd{
			DirPath: "/path",
			Driver:  "mysql",
			Token:   "token",
		}
		p := &mockPrompt{}
		err := setParams(cmd, nil, p)
		require.NoError(t, err)
		require.Equal(t, 0, p.chooseCalled)
		require.Equal(t, 0, p.inputCalled)
	})
	t.Run("run command with no args", func(t *testing.T) {
		cmd := &InitCiCmd{}
		p := &mockPrompt{}
		err := setParams(cmd, []string{"foo", "bar"}, p)
		require.NoError(t, err)
		require.Equal(t, 2, p.chooseCalled)
		require.Equal(t, 1, p.inputCalled)
	})
	t.Run("run command only with token flag", func(t *testing.T) {
		cmd := &InitCiCmd{
			Token: "token",
		}
		p := &mockPrompt{}
		err := setParams(cmd, nil, p)
		require.Equal(t, 2, p.chooseCalled)
		require.Equal(t, 0, p.inputCalled)
		require.NoError(t, err)
	})
}
