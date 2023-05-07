package main

import "github.com/manifoldco/promptui"

// setParams sets the parameters for the init-ci command.
func setParams(cmd *InitCiCmd, dirs []string) error {
	var err error
	// if dir path is not defined we need to ask for the path and the driver
	if cmd.DirPath == "" {
		prompt := promptui.Select{
			Label: "choose migration directory",
			Items: dirs,
		}
		if _, cmd.DirPath, err = prompt.Run(); err != nil {
			return err
		}
		prompt = promptui.Select{
			Label: "choose driver",
			Items: []string{"mysql", "postgres", "mariadb", "sqlite"},
		}
		if _, cmd.Driver, err = prompt.Run(); err != nil {
			return err
		}
	}
	if cmd.Token == "" {
		prompt := promptui.Prompt{
			Label: "enter Atlas Cloud token",
		}
		if cmd.Token, err = prompt.Run(); err != nil {
			return err
		}
	}
	return nil
}
