package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ariga.io/gh-atlas/gen"
	"github.com/1lann/promptui"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/browser"
)

type ContentReader interface {
	// ReadContent of the file at the given path.
	ReadContent(ctx context.Context, path string) (string, error)
}

// setParams sets the parameters for the init-action command.
func (i *InitActionCmd) setParams(ctx context.Context, dirs []string, configs []string, cr ContentReader) error {
	var err error
	if i.DirPath == "" {
		prompt := promptui.Select{
			Label:    "Choose driver",
			HideHelp: true,
			Items:    gen.Drivers,
			Stdin:    i.stdin,
			Templates: &promptui.SelectTemplates{
				Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Driver:" | faint }} {{ . }}`, promptui.IconGood),
			},
		}
		if _, i.Driver, err = prompt.Run(); err != nil {
			return err
		}
		switch {
		case len(dirs) == 0:
			if i.DirPath, err = i.promptForCustomPath(); err != nil {
				return err
			}
		case len(dirs) > 0:
			opts := append(dirs, "provide another path")
			prompt := promptui.Select{
				Label:    "Choose migration directory",
				HideHelp: true,
				Items:    opts,
				Stdin:    i.stdin,
				Templates: &promptui.SelectTemplates{
					Selected: fmt.Sprintf(
						`{{ if ne . "%[1]s" }}{{ "%[2]s" | green }} {{ "%[3]s" | faint }} {{ . }} {{ end }}`,
						"provide another path",
						promptui.IconGood,
						"Migrations directory:",
					),
				},
			}
			if _, i.DirPath, err = prompt.Run(); err != nil {
				return err
			}
		}
		if i.DirPath == "provide another path" {
			if i.DirPath, err = i.promptForCustomPath(); err != nil {
				return err
			}
		}
	}
	if i.ConfigPath == "" && len(configs) > 0 {
		if err = i.setConfigPath(ctx, configs, cr); err != nil {
			return err
		}
	}
	// sqlite has only one schema
	if !i.SchemaScope && i.Driver != "sqlite" {
		prompt := promptui.Select{
			Label: "Do you manage a single schema or multiple? (used to limit the scope of the work done by Atlas)",
			Stdin: i.stdin,
			Items: []string{"single", "multiple"},
			Templates: &promptui.SelectTemplates{
				Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Schema scope:" | faint }} {{ . }}`, promptui.IconGood),
			},
		}
		_, ans, err := prompt.Run()
		if err != nil {
			return err
		}
		switch ans {
		case "single":
			i.SchemaScope = true
		case "multiple":
			i.SchemaScope = false
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
			Templates: &promptui.PromptTemplates{
				Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Atlas Cloud token: " | faint }}`, promptui.IconGood),
			},
		}
		if i.Token, err = prompt.Run(); err != nil {
			return err
		}
	}
	return nil
}

// setConfigPath sets the config path for the init-action command.
func (i *InitActionCmd) setConfigPath(ctx context.Context, configs []string, cr ContentReader) error {
	switch {
	case len(configs) == 1:
		content, err := cr.ReadContent(ctx, configs[0])
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
			envNames := make([]string, 0, len(envs))
			for k := range envs {
				envNames = append(envNames, k)
			}
			prompt := promptui.Select{
				Label:    fmt.Sprintf("Use %q file as config?", configs[0]),
				HideHelp: true,
				Items:    append(envNames, "no"),
				Templates: &promptui.SelectTemplates{
					Active:   "{{ if eq . \"no\" }}▸ No{{ else }}▸ Use with env {{ . | bold }}{{ end }}",
					Inactive: "{{ if eq . \"no\" }}  No{{ else }}  Use with env {{ . | bold }}{{ end }}",
					Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Config env: " | faint }} {{ . }}`, promptui.IconGood),
				},
				Stdin: i.stdin,
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
		prompt := promptui.Select{
			Label:    "Use config file?",
			HideHelp: true,
			Items:    append(configs, "no"),
			Templates: &promptui.SelectTemplates{
				Active:   "{{ if eq . \"no\" }}▸ No{{ else }}▸ Use {{ . | bold }}{{ end }}",
				Inactive: "{{ if eq . \"no\" }}  No{{ else }}  Use {{ . | bold }}{{ end }}",
				Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Config file: " | faint }} {{ . }}`, promptui.IconGood),
			},
			Stdin: i.stdin,
		}
		_, config, err := prompt.Run()
		if err != nil {
			return err
		}
		if config == "no" {
			break
		}
		i.ConfigPath = config
		content, err := cr.ReadContent(ctx, i.ConfigPath)
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
				Templates: &promptui.SelectTemplates{
					Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Config env: " | faint }} {{ . }}`, promptui.IconGood),
				},
			}
			if _, i.ConfigEnv, err = prompt.Run(); err != nil {
				return err
			}
		}
		i.HasDevURL = envs[i.ConfigEnv]
	}
	return nil
}

func (i *InitActionCmd) setDirName(names []string) error {
	var err error
	prompt := promptui.Select{
		Label:    "Choose name of cloud migration directory",
		HideHelp: true,
		Items:    names,
		Stdin:    i.stdin,
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Cloud migration directory:" | faint }} {{ . }}`, promptui.IconGood),
		},
	}
	_, i.DirName, err = prompt.Run()
	return err
}

func (i *InitActionCmd) openURL(url string) error {
	prompt := promptui.Prompt{
		Label:     "Open in browser",
		IsConfirm: true,
		Stdin:     i.stdin,
	}
	if _, err := prompt.Run(); err != nil {
		return err
	}
	return browser.OpenURL(url)
}

func (i *InitActionCmd) promptForCustomPath() (string, error) {
	prompt := promptui.Prompt{
		Label: "Enter the path of the migration directory in your repository",
		Stdin: i.stdin,
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Migrations directory: " | faint }}`, promptui.IconGood),
		},
	}
	return prompt.Run()
}
