package gen

import (
	"bytes"
	"embed"
	"text/template"

	"ariga.io/gh-atlas/atlas"
)

// add logic to generate the code from the template

type Def struct {
	Path    string
	Dialect atlas.Dialect
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

// Generate some code
func Generate(d Def) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "", d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
