package gen

import (
	"bytes"
	_ "embed"
	"fmt"
	"log"
	"text/template"
)

type (
	Driver string
	// Config passed to template parser
	Config struct {
		Path          string
		SecretName    string
		DefaultBranch string
		Driver        string
		Services      string
	}
)

func validateDriver(s string) error {
	switch s {
	case "postgres", "mysql", "mariadb", "sqlite":
		return nil
	default:
		return fmt.Errorf("unknown driver %q", s)
	}
}

var (
	//go:embed services.tmpl
	services_template_resource string
	//go:embed atlas.tmpl
	main_template_resource string
	tmpl                   *template.Template
)

func init() {
	// Based on: https://dev.to/moniquelive/passing-multiple-arguments-to-golang-templates-16h8
	t := template.New("atlas-sync-action").Funcs(template.FuncMap{"args": func(els ...any) []any {
		return els
	}})

	t, err := t.Parse(main_template_resource)

	if err != nil {
		log.Fatalf("Unable to load main template %v", err)
	}

	t, err = t.Parse(services_template_resource)

	if err != nil {
		log.Fatalf("Unable to load services template %v", err)
	}

	tmpl = t
}

// Generate the content of the atlas ci lint yaml.
func Generate(cfg *Config) ([]byte, error) {
	if err := validateDriver(cfg.Driver); err != nil {
		return nil, err
	}
	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, "atlas-sync-action", cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
