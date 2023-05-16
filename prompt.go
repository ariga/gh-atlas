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
			Label: "choose migration directory",
			Items: dirs,
			Stdin: i.stdin,
		}
		if _, i.DirPath, err = prompt.Run(); err != nil {
			return err
		}
		prompt = promptui.Select{
			Label: "choose driver",
			Items: []string{"mysql", "postgres", "mariadb", "sqlite"},
			Stdin: i.stdin,
		}
		if _, i.Driver, err = prompt.Run(); err != nil {
			return err
		}
	}
	if i.Token == "" {
		prompt := promptui.Prompt{
			Label: "enter Atlas Cloud token",
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
