package main

import (
	"github.com/AlecAivazis/survey/v2"
)

type (
	// prompter is an interface for prompting the user for input.
	prompter interface {
		// ask user to choose from a list of options, return the selected option
		choose(msg string, opts []string) (string, error)
		// ask user for input
		input(msg string) (string, error)
	}
	// stdPrompt is the default prompter to the standard I/O.
	stdPrompt struct{}
)

func (s *stdPrompt) choose(msg string, opts []string) (string, error) {
	var res string
	prompt := &survey.Select{
		Message: msg,
		Options: opts,
	}
	if err := survey.AskOne(prompt, &res); err != nil {
		return "", err
	}
	return res, nil
}

func (s *stdPrompt) input(msg string) (string, error) {
	var res string
	prompt := &survey.Input{
		Message: msg,
	}
	if err := survey.AskOne(prompt, &res); err != nil {
		return "", err
	}
	return res, nil
}

// setParams sets the parameters for the init-ci command.
func setParams(cmd *InitCiCmd, dirs []string, p prompter) error {
	if p == nil {
		p = &stdPrompt{}
	}
	var err error
	// if dir path is not defined we need to ask for the path and the driver
	if cmd.DirPath == "" {
		if cmd.DirPath, err = p.choose("choose migration directory", dirs); err != nil {
			return err
		}
		if cmd.Driver, err = p.choose("choose driver", []string{"mysql", "postgres", "mariadb", "sqlite"}); err != nil {
			return err
		}
	}
	if cmd.Token == "" {
		if cmd.Token, err = p.input("enter Atlas Cloud token"); err != nil {
			return err
		}
	}
	return nil
}
