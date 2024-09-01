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
	if err := i.setDirPath(dirs); err != nil {
		return err
	}
	if i.ConfigPath == "" && len(configs) > 0 {
		if err := i.setConfigPath(ctx, configs, cr); err != nil {
			return err
		}
	}
	if !i.HasDevURL && i.Driver == "" {
		if err := i.setDriver(); err != nil {
			return err
		}
	}
	if err := i.setSchemaScope(); err != nil {
		return err
	}
	if err := i.setToken(); err != nil {
		return err
	}
	return nil
}

func (i *InitActionCmd) setDriver() error {
	prompt := promptui.Select{
		Label:    "Choose driver",
		HideHelp: true,
		Items:    gen.Drivers,
		Stdin:    i.stdin,
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Driver:" | faint }} {{ . }}`, promptui.IconGood),
		},
	}
	_, driver, err := prompt.Run()
	i.Driver = driver
	return err
}

func (i *InitActionCmd) setDirPath(dirs []string) error {
	if i.DirPath != "" {
		return nil
	}
	var err error
	switch {
	case len(dirs) == 0:
		i.DirPath, err = i.promptForCustomPath()
	case len(dirs) > 0:
		i.DirPath, err = i.chooseDirPath(dirs)
	}
	return err
}

func (i *InitActionCmd) chooseDirPath(dirs []string) (string, error) {
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
	_, path, err := prompt.Run()
	if err != nil {
		return "", err
	}
	if path == "provide another path" {
		return i.promptForCustomPath()
	}
	return path, nil
}

func (i *InitActionCmd) setSchemaScope() error {
	// sqlite has only one schema
	if i.SchemaScope || i.Driver == "sqlite" {
		return nil
	}
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
	i.SchemaScope = ans == "single"
	return nil
}

func (i *InitActionCmd) setToken() error {
	if i.Token != "" {
		return nil
	}
	prompt := promptui.Prompt{
		Label: "Enter Atlas Cloud token",
		Stdin: i.stdin,
		Mask:  '*',
		Validate: func(s string) error {
			if strings.TrimSpace(s) == "" {
				return errors.New("token cannot be empty")
			}
			return nil
		},
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Atlas Cloud token: " | faint }}`, promptui.IconGood),
		},
	}
	token, err := prompt.Run()
	i.Token = token
	return err
}

func (i *InitActionCmd) setConfigPath(ctx context.Context, configs []string, cr ContentReader) error {
	switch len(configs) {
	case 0:
		return nil
	case 1:
		return i.handleSingleConfig(ctx, configs[0], cr)
	default:
		return i.handleMultipleConfigs(ctx, configs, cr)
	}
}

func (i *InitActionCmd) handleSingleConfig(ctx context.Context, config string, cr ContentReader) error {
	envs, err := i.getEnvs(ctx, config, cr)
	if err != nil {
		return err
	}
	if len(envs) == 0 {
		return nil
	}
	return i.chooseEnv(config, envs)
}

func (i *InitActionCmd) handleMultipleConfigs(ctx context.Context, configs []string, cr ContentReader) error {
	config, err := i.chooseConfig(configs)
	if err != nil || config == "no" {
		return err
	}
	i.ConfigPath = config
	envs, err := i.getEnvs(ctx, config, cr)
	if err != nil {
		return err
	}
	return i.setConfigEnv(envs)
}

func (i *InitActionCmd) getEnvs(ctx context.Context, path string, cr ContentReader) (map[string]bool, error) {
	content, err := cr.ReadContent(ctx, path)
	if err != nil {
		return nil, err
	}
	file, diags := hclparse.NewParser().ParseHCL([]byte(content), "atlas.hcl")
	if len(diags) > 0 {
		return nil, fmt.Errorf("failed to parse %s: %s", path, diags.Error())
	}
	envs := make(map[string]bool)
	for _, b := range file.Body.(*hclsyntax.Body).Blocks {
		if b.Type == "env" {
			_, hasDev := b.Body.Attributes["dev"]
			envs[b.Labels[0]] = hasDev
		}
	}
	return envs, nil
}

func (i *InitActionCmd) chooseConfig(configs []string) (string, error) {
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
	return config, err
}

func (i *InitActionCmd) chooseEnv(cfgFile string, envs map[string]bool) error {
	envNames := make([]string, 0, len(envs))
	for k := range envs {
		envNames = append(envNames, k)
	}
	var prompt promptui.Select
	if cfgFile != "" {
		prompt = promptui.Select{
			Label:    fmt.Sprintf("Use %q file as config?", cfgFile),
			HideHelp: true,
			Items:    append(envNames, "no"),
			Templates: &promptui.SelectTemplates{
				Active:   "{{ if eq . \"no\" }}▸ No{{ else }}▸ Use with env {{ . | bold }}{{ end }}",
				Inactive: "{{ if eq . \"no\" }}  No{{ else }}  Use with env {{ . | bold }}{{ end }}",
				Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Config env: " | faint }} {{ . }}`, promptui.IconGood),
			},
			Stdin: i.stdin,
		}
	} else {
		prompt = promptui.Select{
			Label:    "Choose an environment",
			HideHelp: true,
			Items:    envNames,
			Stdin:    i.stdin,
			Templates: &promptui.SelectTemplates{
				Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Config env: " | faint }} {{ . }}`, promptui.IconGood),
			},
		}
	}
	_, env, err := prompt.Run()
	if err != nil {
		return err
	}
	if env == "no" {
		return nil
	}
	if cfgFile != "" {
		i.ConfigPath = cfgFile
	}
	i.ConfigEnv = env
	i.HasDevURL = envs[env]
	return nil
}

func (i *InitActionCmd) setConfigEnv(envs map[string]bool) error {
	switch len(envs) {
	case 0:
		return nil
	case 1:
		for k := range envs {
			i.ConfigEnv = k
		}
		i.HasDevURL = envs[i.ConfigEnv]
		return nil
	default:
		return i.chooseEnv("", envs)
	}
}

func (i *InitActionCmd) setDirName(names []string) error {
	prompt := promptui.Select{
		Label:    "Choose name of cloud migration directory",
		HideHelp: true,
		Items:    names,
		Stdin:    i.stdin,
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Cloud migration directory:" | faint }} {{ . }}`, promptui.IconGood),
		},
	}
	_, name, err := prompt.Run()
	i.DirName = name
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
