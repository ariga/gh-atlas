package gen

import (
	"bytes"
	"embed"
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
	//go:embed *.tmpl
	files embed.FS

	tmpl = template.Must(template.New("atlas-sync-action").
		Funcs(argsFunc()).
		ParseFS(files, "*.tmpl"))
)

func argsFunc() template.FuncMap {
	return template.FuncMap{"args": func(els ...any) []any {
		return els
	}}
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
