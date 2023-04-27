package main

import "github.com/AlecAivazis/survey/v2"

// ask user to choose from a list of options, return the selected option
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

// ask user for input
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
