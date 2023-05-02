package main

import (
	"github.com/AlecAivazis/survey/v2"
)

// setParams sets the parameters for the init-ci command.
func setParams(cmd *InitCiCmd, dirs []string) error {
	var err error
	// if dir path is not defined we need to ask for the path and the driver
	if cmd.DirPath == "" {
		if cmd.DirPath, err = choose("choose migration directory", dirs); err != nil {
			return err
		}
		if cmd.Driver, err = choose("choose driver", []string{"mysql", "postgres", "mariadb", "sqlite"}); err != nil {
			return err
		}
	}
	if cmd.Token == "" {
		if cmd.Token, err = input("enter Atlas Cloud token"); err != nil {
			return err
		}
	}
	return nil
}

func choose(msg string, opts []string) (string, error) {
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

func input(msg string) (string, error) {
	var res string
	prompt := &survey.Input{
		Message: msg,
	}
	if err := survey.AskOne(prompt, &res); err != nil {
		return "", err
	}
	return res, nil
}
