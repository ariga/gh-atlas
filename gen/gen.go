package gen

import (
	"bytes"
	_ "embed"
	"fmt"
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
	t := template.New("atlas-sync-action").Funcs(template.FuncMap{"args": func(els ...any) []any {
		return els
	}})

	t = template.Must(t.Parse(main_template_resource))
	tmpl = template.Must(t.Parse(services_template_resource))
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
