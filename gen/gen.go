package gen

import (
	"bytes"
	_ "embed"
	"fmt"
	"text/template"
)

type (
	// Config passed to template parser
	Config struct {
		Path          string
		SecretName    string
		DefaultBranch string
		Driver        string
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
	//go:embed atlas.tmpl
	resource string
	tmpl     *template.Template
)

func init() {
	t, err := template.New("atlas.tmpl").Parse(resource)
	if err != nil {
		panic(err)
	}
	tmpl = t
}

// Generate the content of the atlas ci lint yaml.
func Generate(cfg *Config) ([]byte, error) {
	if err := validateDriver(cfg.Driver); err != nil {
		return nil, err
	}
	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, "atlas.tmpl", cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
