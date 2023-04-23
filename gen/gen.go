package gen

import (
	"bytes"
	"embed"
	"errors"
	"text/template"
)

type (
	Dialect string
	// Config passed to template parser
	Config struct {
		Path          string
		DefaultBranch string
		Dialect       Dialect
	}
)

const (
	Postgres Dialect = "postgres"
	MySQL    Dialect = "mysql"
	MariaDB  Dialect = "maria"
	SQLite   Dialect = "sqlite"
)

// GetDialect returns the dialect from string.
func GetDialect(s string) (Dialect, error) {
	switch s {
	case "postgres":
		return Postgres, nil
	case "mysql":
		return MySQL, nil
	case "mariadb":
		return MariaDB, nil
	case "sqlite":
		return SQLite, nil
	default:
		return "", errors.New("unknown database dialect")
	}
}

var (
	//go:embed *.tmpl
	resource embed.FS
	tmpl     *template.Template
)

func init() {
	t := template.New("")
	t, err := t.ParseFS(resource, "*.tmpl")
	if err != nil {
		panic(err)
	}
	tmpl = t
}

// Generate the content of the atlas ci lint yaml.
func Generate(cfg *Config) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, "atlas.tmpl", cfg); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
