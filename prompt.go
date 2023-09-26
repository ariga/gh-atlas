package main

import (
	"errors"
	"strings"

	"github.com/1lann/promptui"
)

// setParams sets the parameters for the init-action command.
func (i *InitActionCmd) setParams(dirs []string) error {
	var err error
	if i.DirPath == "" {
		prompt := promptui.Select{
			Label: "Choose migration directory",
			Items: dirs,
			Stdin: i.stdin,
		}
		if _, i.DirPath, err = prompt.Run(); err != nil {
			return err
		}
		prompt = promptui.Select{
			Label: "Choose driver",
			Items: []string{"mysql", "postgres", "mariadb", "sqlite"},
			Stdin: i.stdin,
		}
		if _, i.Driver, err = prompt.Run(); err != nil {
			return err
		}
	}
	if i.Token == "" {
		prompt := promptui.Prompt{
			Label: "Enter Atlas Cloud token",
			Stdin: i.stdin,
			Mask:  '*',
			Validate: func(s string) error {
				if strings.Trim(s, " ") == "" {
					return errors.New("token cannot be empty")
				}
				return nil
			},
		}
		if i.Token, err = prompt.Run(); err != nil {
			return err
		}
	}
	return nil
}

func (i *InitActionCmd) setDirName(names []string) error {
	var err error
	prompt := promptui.Select{
		Label: "Choose action",
		Items: []string{"Use existing directory from the Cloud", "Create new directory"},
		Stdin: i.stdin,
	}
	_, rsp, err := prompt.Run()
	if err != nil {
		return err
	}
	if rsp == "Create new directory" {
		return nil
	}
	prompt = promptui.Select{
		Label: "Choose name of cloud migration directory",
		Items: names,
		Stdin: i.stdin,
	}
	if _, i.DirName, err = prompt.Run(); err != nil {
		return err
	}
	return nil
}
