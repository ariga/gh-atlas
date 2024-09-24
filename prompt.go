package main

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ariga.io/gh-atlas/cloudapi"
	"ariga.io/gh-atlas/gen"
	"github.com/1lann/promptui"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/pkg/browser"
)

type flowType string

const (
	Versioned   flowType = "versioned"
	Declarative flowType = "declarative"

	UnnamedEnv = "OTHER"
)

// setParams sets the parameters for the init-action command.
func (i *InitActionCmd) setParams(ctx context.Context, re RepoExplorer, cloud cloudapi.API) error {
	var (
		err  error
		repo *cloudapi.Repo
	)
	if repo, err = i.selectAtlasRepo(ctx, cloud); err != nil {
		return err
	}
	if err = i.initializeFlow(repo); err != nil {
		return err
	}
	if i.env.Path == "" {
		configs, err := re.ConfigFiles(ctx)
		if err != nil {
			return err
		}
		if err := i.setAtlasConfig(ctx, configs, re); err != nil {
			return err
		}
	}
	switch i.flow {
	case Versioned:
		if i.DirName == "" && repo != nil {
			i.DirName = repo.Slug
		}
		fmt.Printf("%s %s %s\n",
			promptui.IconGood,
			promptui.Styler(promptui.FGFaint)("Target migrations directory name:"),
			i.DirName)
		dirs, err := re.MigrationDirectories(ctx)
		if err != nil {
			return err
		}
		if err := i.setDirPath(dirs); err != nil {
			return err
		}
	case Declarative:
		if i.To == "" && repo != nil {
			i.cloudRepo = repo.Slug
		}
		if !i.env.HasURL && !i.env.HasRepoName {
			if err := i.setCurrentState(); err != nil {
				return err
			}
		}
		if !i.env.HasSchemaSrc {
			if err := i.setDesiredState(); err != nil {
				return err
			}
		}
		if err := i.setSetupSchemaApply(); err != nil {
			return err
		}
	}
	if repo != nil {
		i.driver = repo.Driver
	}
	if !i.env.HasDevURL && i.driver == "" {
		if err := i.setDriver(); err != nil {
			return err
		}
	}
	if err := i.setSchemaScope(); err != nil {
		return err
	}
	// Params can be set by flags or prompts, so validate them here
	return i.validateParams()
}

func (i *InitActionCmd) initializeFlow(repo *cloudapi.Repo) error {
	var repoType cloudapi.RepoType
	if repo != nil {
		repoType = repo.Type
	}
	switch {
	case repoType == cloudapi.SchemaType || i.To != "":
		i.flow = Declarative
	case repoType == cloudapi.DirectoryType || i.DirName != "":
		i.flow = Versioned
	default:
		return fmt.Errorf("cannot infer flow from repo type %q", repoType)
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
	i.driver = driver
	return err
}

func (i *InitActionCmd) selectAtlasRepo(ctx context.Context, cloud cloudapi.API) (*cloudapi.Repo, error) {
	repos, err := cloud.Repos(ctx)
	if err != nil {
		return nil, err
	}
	// if i.To or i.DirName are set (by flags),
	// it means that the user wants to use a specific schema state or migration directory
	// we'll search for the repo with the same name
	// if found, return it
	if i.To != "" || i.DirName != "" {
		found := false
		for _, r := range repos {
			if r.URL == i.To || r.Slug == i.DirName {
				return &r, nil
			}
		}
		// otherwise, the a specific repo is set, but not found, return an error
		if !found {
			return nil, errors.New("no repository with given name found")
		}
	}
	if len(repos) == 0 {
		return nil, fmt.Errorf("no repositories found")
	}
	if len(repos) == 1 {
		return &repos[0], nil
	}
	repoTitles := make([]string, 0, len(repos))
	byTitle := make(map[string]cloudapi.Repo)
	for _, r := range repos {
		repoTitles = append(repoTitles, r.Title)
		byTitle[r.Title] = r
	}
	prompt := promptui.Select{
		Label: "Select an Atlas Cloud Repository",
		Items: repoTitles,
		Stdin: i.stdin,
		Templates: &promptui.SelectTemplates{
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Selected repository:" | faint }} {{ . }}`, promptui.IconGood),
		},
	}
	_, t, err := prompt.Run()
	if err != nil {
		return nil, err
	}
	selected, ok := byTitle[t]
	if !ok {
		return nil, fmt.Errorf("no repository with name %q found", t)
	}
	return &selected, nil
}

func (i *InitActionCmd) setDesiredState() error {
	if i.To != "" {
		return nil
	}
	prompt := promptui.Prompt{
		Label: "Enter a URL of the desired schema state",
		Stdin: i.stdin,
		Validate: func(input string) error {
			if len(i.To) == 0 && strings.TrimSpace(input) == "" {
				return errors.New("at least one URL is required for desired schema state")
			}
			return nil
		},
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Desired schema state URL: " | faint }}`, promptui.IconGood),
		},
	}
	f, err := prompt.Run()
	if err != nil {
		return err
	}
	i.To = f
	return nil
}

func (i *InitActionCmd) setCurrentState() error {
	if i.From != "" {
		return nil
	}
	prompt := promptui.Prompt{
		Label: "Enter a URL of the current schema state",
		Stdin: i.stdin,
		Validate: func(input string) error {
			if len(i.From) == 0 && strings.TrimSpace(input) == "" {
				return errors.New("at least one URL is required for current schema state")
			}
			return nil
		},
		Templates: &promptui.PromptTemplates{
			Valid:   fmt.Sprintf(`{{ "%s" | green }} {{ . | faint }} {{": " | faint }}`, promptui.IconGood),
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Current schema state URL: " | faint }}`, promptui.IconGood),
		},
	}
	f, err := prompt.Run()
	if err != nil {
		return err
	}
	i.From = f
	return nil
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
	if i.SchemaScope || i.driver == "SQLITE" {
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

func (i *InitActionCmd) setAtlasConfig(ctx context.Context, configs []string, cr RepoExplorer) error {
	if len(configs) == 0 {
		return nil
	}
	return i.handleAtlasConfig(ctx, configs, cr)
}

func (i *InitActionCmd) handleAtlasConfig(ctx context.Context, configs []string, cr RepoExplorer) error {
	config, err := i.chooseConfig(configs)
	if err != nil || config == "no" {
		return err
	}
	i.env.Path = config
	envs, err := i.getEnvs(ctx, config, cr)
	if err != nil {
		return err
	}
	if len(envs) == 0 {
		return nil
	}
	return i.chooseEnv(envs, config)
}

func (i *InitActionCmd) getEnvs(ctx context.Context, path string, cr RepoExplorer) (envs map[string]gen.Env, err error) {
	content, err := cr.ReadContent(ctx, path)
	if err != nil {
		return nil, err
	}
	file, diags := hclparse.NewParser().ParseHCL([]byte(content), "atlas.hcl")
	if len(diags) > 0 {
		return nil, fmt.Errorf("failed to parse %s: %s", path, diags.Error())
	}
	envs = make(map[string]gen.Env)
	for _, b := range file.Body.(*hclsyntax.Body).Blocks {
		if b.Type == "env" {
			var (
				name         string
				hasDevURL    bool
				hasURL       bool
				hasSchemaSrc bool
				hasRepoName  bool
			)
			if len(b.Labels) == 0 {
				//TODO: fix, it may conflict with other envs names
				name = UnnamedEnv
			} else {
				name = b.Labels[0]
			}
			_, hasDevURL = b.Body.Attributes["dev"]
			_, hasURL = b.Body.Attributes["url"]
			for _, bb := range b.Body.Blocks {
				if bb.Type == "schema" {
					_, hasSchemaSrc = bb.Body.Attributes["src"]
				}
				for _, bbb := range bb.Body.Blocks {
					if bbb.Type == "repo" {
						_, hasRepoName = bbb.Body.Attributes["name"]
					}
				}
			}
			envs[name] = gen.Env{
				Name:         name,
				HasDevURL:    hasDevURL,
				HasURL:       hasURL,
				HasSchemaSrc: hasSchemaSrc,
				HasRepoName:  hasRepoName,
				Path:         path,
			}
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

func (i *InitActionCmd) chooseEnv(envs map[string]gen.Env, config string) error {
	envNames := make([]string, 0, len(envs))
	hasUnnamedEnv := false
	for k := range envs {
		envNames = append(envNames, k)
		if k == UnnamedEnv {
			hasUnnamedEnv = true
		}
	}
	if len(envNames) == 1 && hasUnnamedEnv {
		name, err := i.promptForEnvName()
		if err != nil {
			return err
		}
		var env gen.Env
		for _, v := range envs {
			env = v
		}
		env.Name = name
		i.env = env
		return nil
	}
	if len(envNames) == 1 {
		i.env = envs[envNames[0]]
		return nil
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
	_, ans, err := prompt.Run()
	if err != nil {
		return err
	}
	if ans == UnnamedEnv {
		name, err := i.promptForEnvName()
		if err != nil {
			return err
		}
		var env gen.Env
		for _, v := range envs {
			env = v
		}
		env.Name = name
		i.env = env
		return nil
	}
	i.env = envs[ans]
	return nil
}

func (i *InitActionCmd) promptForEnvName() (string, error) {
	prompt := promptui.Prompt{
		Label: "Enter the name of the env block",
		Stdin: i.stdin,
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Env block name: " | faint }}`, promptui.IconGood),
		},
	}
	return prompt.Run()
}

func (i *InitActionCmd) openURL(url string) error {
	prompt := promptui.Prompt{
		Label:     "Open in browser",
		IsConfirm: true,
		Stdin:     i.stdin,
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Open in browser: " | faint }}`, promptui.IconGood),
		},
	}
	_, err := prompt.Run()
	if err != nil {
		// https://github.com/manifoldco/promptui/issues/81
		// the promptui library generates ErrAbort if response is 'n'
		if errors.Is(err, promptui.ErrAbort) {
			return nil
		}
		return err
	}
	return browser.OpenURL(url)
}

func (i *InitActionCmd) promptForCustomPath() (string, error) {
	prompt := promptui.Prompt{
		Label: "Enter the path of the migration directory in your repository",
		Stdin: i.stdin,
		Templates: &promptui.PromptTemplates{
			Success: fmt.Sprintf(`{{ "%s" | green }} {{ "Migrations directory path: " | faint }}`, promptui.IconGood),
		},
	}
	return prompt.Run()
}

func (i *InitActionCmd) setSetupSchemaApply() error {
	if i.SetupSchemaApply != nil {
		// already set by flag
		return nil
	}
	prompt := promptui.Select{
		Label: "Do you want to setup the `schema apply` action?",
		Stdin: i.stdin,
		Items: []string{"yes", "no"},
		Templates: &promptui.SelectTemplates{
			Active:   "{{ if eq . \"no\" }}▸ No{{ else }}▸ Yes{{ end }}",
			Inactive: "{{ if eq . \"no\" }}  No{{ else }}  Yes{{ end }}",
			Selected: fmt.Sprintf(`{{ "%s" | green }} {{ "Setup schema apply: " | faint }} {{ . }}`, promptui.IconGood),
		},
	}
	_, setup, err := prompt.Run()
	if err != nil {
		return err
	}
	s := setup == "yes"
	i.SetupSchemaApply = &s
	return nil
}

func (i *InitActionCmd) validateParams() error {
	switch i.flow {
	case Versioned:
		if i.DirPath == "" {
			return errors.New("dirpath is required for versioned flow")
		}
		if i.DirName == "" {
			return errors.New("dirname is required for versioned flow")
		}
		if i.From != "" || i.To != "" {
			return errors.New("from and to are not applicable for versioned flow")
		}
	case Declarative:
		if i.DirPath != "" || i.DirName != "" {
			return errors.New("dirpath and dirname are not applicable for declarative flow")
		}
	}
	return nil
}
