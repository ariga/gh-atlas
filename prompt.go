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
		if len(dirs) == 0 {
			return errors.New("no migration directories found in the repository, please enter dir path manually by running:\n" +
			`gh atlas init-action "dir/path/migrations"` )
		}
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
	if len(names) == 1 {
		i.DirName = names[0]
		return nil
	}
	var err error
	prompt := promptui.Select{
		Label: "Choose name of cloud migration directory",
		Items: names,
		Stdin: i.stdin,
	}
	_, i.DirName, err = prompt.Run()
	return err
}
