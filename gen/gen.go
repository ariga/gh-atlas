package gen

import (
	"bytes"
	"embed"
	"text/template"
)

type (
	Dialect string
	Def     struct {
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

// Generate the atlas ci yaml file content.
func Generate(d *Def) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	if err := tmpl.ExecuteTemplate(b, "atlas.tmpl", d); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
