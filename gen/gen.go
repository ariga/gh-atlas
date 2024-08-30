package gen

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"slices"
	"text/template"
)

type (
	Driver string
	// Config passed to template parser
	Config struct {
		Flow          string
		Path          string
		DirName       string
		SecretName    string
		DefaultBranch string
		Driver        string
		ConfigPath    string
		Env           string
		CreateDevURL  bool
		SchemaScope   bool
		From          []string
		To            []string
	}
)

var (
	Drivers = []string{"mysql", "postgres", "postgis", "mariadb", "sqlite", "mssql", "clickhouse"}
	Flows   = []string{"migrate", "schema"}
	//go:embed *.tmpl
	files embed.FS

	tmpl = template.Must(template.New("atlas-sync-action").
		ParseFS(files, "*.tmpl"))
)

// Generate the content of the atlas ci lint yaml.
func Generate(cfg *Config) ([]byte, error) {
	if err := validateDriver(cfg.Driver); err != nil {
		return nil, err
	}
	if err := validateFlow(cfg.Flow); err != nil {
		return nil, err
	}
	b := bytes.NewBuffer(nil)

	if err := tmpl.ExecuteTemplate(b, "atlas.tmpl", cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func validateDriver(s string) error {
	if !slices.Contains(Drivers, s) {
		return fmt.Errorf("unknown driver %q", s)
	}
	return nil
}

func validateFlow(s string) error {
	if !slices.Contains(Flows, s) {
		return fmt.Errorf("unknown flow %q", s)
	}
	return nil
}
