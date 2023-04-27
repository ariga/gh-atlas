package main

import (
	"github.com/AlecAivazis/survey/v2"
)

// ask user to select from a list of options, return the selected option
func ask(label string, opts []string) (string, error) {
	var res string
	prompt := &survey.Select{
		Message: label,
		Options: opts,
	}
	if err := survey.AskOne(prompt, &res); err != nil {
		return "", err
	}
	return res, nil
}
