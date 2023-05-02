package main

import (
	"os"
	"testing"

	"github.com/AlecAivazis/survey/v2/terminal"
	"github.com/Netflix/go-expect"
	pseudotty "github.com/creack/pty"
	"github.com/hinshun/vt10x"
	"github.com/stretchr/testify/require"
)

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
				Driver:  "driver",
				Token:   "token",
			},
			prompt: func(c *expect.Console) {
				_, err := c.ExpectEOF()
				require.NoError(t, err)
			},
			expected: &InitCiCmd{
				DirPath: "path",
				Driver:  "driver",
				Token:   "token",
			},
		},
		{
			name: "no token flag supplied",
			cmd: &InitCiCmd{
				DirPath: "path",
				Driver:  "driver",
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
				Driver:  "driver",
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
				DirPath: "dir2",
				Driver:  "postgres",
				Token:   "token",
			},
		},
	}
	{
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				RunTest(t, tt.prompt, func() error {
					return tt.cmd.Run(&Context{Testing: true})
				})
			})
			require.Equal(t, tt.expected, tt.cmd)
		}
	}
}

// RunTest runs a given test with expected I/O prompt.
func RunTest(t *testing.T, prompt func(*expect.Console), test func() error) {
	pty, tty, err := pseudotty.Open()
	require.NoError(t, err)
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
