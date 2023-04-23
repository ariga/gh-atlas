package gen

import (
	"bytes"
	"embed"
	"errors"
	"text/template"
)

type (
	Dialect string
	// Def is the definition passed to template parser
	Def struct {
		Path          string
		DefaultBranch string
		Dialect       Dialect
	}
)

const (
	Postgres Dialect = "postgres"
	Mysql    Dialect = "mysql"
	Mariadb  Dialect = "maria"
	Sqlite   Dialect = "sqlite"
)

// GetDialect returns the dialect from string.
func GetDialect(s string) (Dialect, error) {
	switch s {
	case "postgres":
		return Postgres, nil
	case "mysql":
		return Mysql, nil
	case "mariadb":
		return Mariadb, nil
	case "sqlite":
		return Sqlite, nil
	default:
		return "", errors.New("unknown database dialect")
	}
}

var (
	//go:embed templates/*
	resource embed.FS
	tmpl     *template.Template
)

func init() {
	t := template.New("")
	t, err := t.ParseFS(resource, "templates/*.tmpl")
	if err != nil {
		panic(err)
	}
	tmpl = t
}

// Generate the content of the atlas ci lint yaml.
func Generate(d *Def) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, "atlas.tmpl", d); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
