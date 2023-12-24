package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/1lann/promptui"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// setParams sets the parameters for the init-action command.
func (i *InitActionCmd) setParams(ctx context.Context, repo *Repository) error {
	dirs, err := repo.MigrationDirectories(ctx)
	if err != nil {
		return err
	}
	configs, err := repo.ConfigFiles(ctx)
	if err != nil {
		return err
	}
	if i.DirPath == "" {
		prompt := promptui.Select{
			Label: "Choose driver",
			Items: []string{"mysql", "postgres", "mariadb", "sqlite"},
			Stdin: i.stdin,
		}
		if _, i.Driver, err = prompt.Run(); err != nil {
			return err
		}
		switch {
		case len(dirs) == 0:
			prompt := promptui.Prompt{
				Label: "Enter the path of the migration directory in your repository",
				Stdin: i.stdin,
			}
			if i.DirPath, err = prompt.Run(); err != nil {
				return err
			}
		case len(dirs) > 0:
			opts := append(dirs, "provide another path")
			prompt := promptui.Select{
				Label: "Choose migration directory",
				Items: opts,
				Stdin: i.stdin,
			}
			if _, i.DirPath, err = prompt.Run(); err != nil {
				return err
			}
		}
		if i.DirPath == "provide another path" {
			prompt := promptui.Prompt{
				Label: "Enter the path of the migration directory in your repository",
				Stdin: i.stdin,
			}
			if i.DirPath, err = prompt.Run(); err != nil {
				return err
			}
		}
	}
	if i.ConfigPath == "" && len(configs) > 0 {
		prompt := promptui.Select{
			Label: "Choose atlas.hcl config file path to use",
			Items: configs,
			Stdin: i.stdin,
		}
		if _, i.ConfigPath, err = prompt.Run(); err != nil {
			return err
		}
		content, err := repo.ReadContent(ctx, i.ConfigPath)
		if err != nil {
			return err
		}
		file, diags := hclparse.NewParser().ParseHCL([]byte(content), "atlas.hcl")
		if len(diags) > 0 {
			return fmt.Errorf("failed to parse %s: %s", i.ConfigPath, diags.Error())
		}
		var envs = make(map[string]bool)
		for _, b := range file.Body.(*hclsyntax.Body).Blocks {
			if b.Type == "env" {
				_, hasDev := b.Body.Attributes["dev"]
				envs[b.Labels[0]] = hasDev
			}
		}
		switch {
		case len(envs) == 0:
			return fmt.Errorf("no env blocks found in %s", i.ConfigPath)
		case len(envs) > 0:
			envNames := make([]string, 0, len(envs))
			for k := range envs {
				envNames = append(envNames, k)
			}
			prompt := promptui.Select{
				Label: "Choose environment to use from the config file",
				Items: envNames,
				Stdin: i.stdin,
			}
			if _, i.ConfigEnv, err = prompt.Run(); err != nil {
				return err
			}
		}
		i.HasDevURL = envs[i.ConfigEnv]
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
