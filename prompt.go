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
			Label:    "Choose driver",
			HideHelp: true,
			Items:    []string{"mysql", "postgres", "mariadb", "sqlite"},
			Stdin:    i.stdin,
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
				Label:    "Choose migration directory",
				HideHelp: true,
				Items:    opts,
				Stdin:    i.stdin,
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
		switch {
		case len(configs) == 1:
			content, err := repo.ReadContent(ctx, configs[0])
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
			if len(envs) > 0 {
				opts := []string{"no"}
				envNames := make([]string, 0, len(envs))
				for k := range envs {
					envNames = append(envNames, k)
				}
				opts = append(opts, envNames...)
				prompt := promptui.Select{
					Label:    fmt.Sprintf("Use %q file as config?", configs[0]),
					HideHelp: true,
					Items:    opts,
					Templates: &promptui.SelectTemplates{
						Active:   "{{ if eq . \"no\" }}▸ no{{ else }}▸ use with env {{ . | bold }}{{ end }}",
						Inactive: "{{ if eq . \"no\" }}  no{{ else }}  use with env {{ . | bold }}{{ end }}",
					},
					Stdin:    i.stdin,
				}
				_, env, err := prompt.Run()
				if err != nil {
					return err
				}
				if env == "no" {
					break
				}
				i.ConfigPath = configs[0]
				i.ConfigEnv = env
				i.HasDevURL = envs[env]
			}

		case len(configs) > 1:
			opts := []string{"no"}
			opts = append(opts, configs...)
			prompt := promptui.Select{
				Label:    "Use config file?",
				HideHelp: true,
				Items:    opts,
				Templates: &promptui.SelectTemplates{
					Active:   "{{ if eq . \"no\" }}▸ no{{ else }}▸ use {{ . | bold }}{{ end }}",
					Inactive: "{{ if eq . \"no\" }}  no{{ else }}  use {{ . | bold }}{{ end }}",
				},
				Stdin:    i.stdin,
			}
			_, config, err := prompt.Run()
			if err != nil {
				return err
			}
			if config == "no" {
				break
			}
			i.ConfigPath = config
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
				break
			case len(envs) == 1:
				for k := range envs {
					i.ConfigEnv = k
				}
			case len(envs) > 1:
				envNames := make([]string, 0, len(envs))
				for k := range envs {
					envNames = append(envNames, k)
				}
				prompt := promptui.Select{
					Label:    "Choose an environment",
					HideHelp: true,
					Items:    envNames,
					Stdin:    i.stdin,
				}
				if _, i.ConfigEnv, err = prompt.Run(); err != nil {
					return err
				}
			}
			i.HasDevURL = envs[i.ConfigEnv]

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
		Label:    "Choose name of cloud migration directory",
		HideHelp: true,
		Items:    names,
		Stdin:    i.stdin,
	}
	_, i.DirName, err = prompt.Run()
	return err
}
