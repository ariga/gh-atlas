package gen

import (
	"bytes"
	"embed"
	"text/template"
)

type (
	Env struct {
		Name string
		// HasDevURL is true if the env block has dev attribute
		HasDevURL bool
		// HasURL is true if the env block has url attribute
		HasURL bool
		// Path is the path of the file containing the env block
		Path string
		// HasSchemaSrc is true if the env block has schema.src attribute
		HasSchemaSrc bool
		// HasRepoName is true if the schema block has repo.name attribute
		HasRepoName bool
	}
	// Config passed to template parser
	Config struct {
		Flow             string
		From             string
		To               string
		Path             string
		DirName          string
		SecretName       string
		DefaultBranch    string
		Driver           string
		Env              Env
		CreateDevURL     bool
		SchemaScope      bool
		CloudRepo        string
		SetupSchemaApply bool
	}
)

var (
	Drivers = []string{"MYSQL", "POSTGRESQL", "MARIADB", "SQLITE", "SQLSERVER", "CLICKHOUSE"}
	//go:embed *.tmpl
	files embed.FS
	tmpl  = template.Must(template.New("atlas-sync-action").
		ParseFS(files, "*.tmpl"))
)

// Generate the content of the atlas ci lint yaml.
func Generate(cfg *Config) ([]byte, error) {
	b := bytes.NewBuffer(nil)
	switch cfg.Flow {
	case "versioned":
		if err := tmpl.ExecuteTemplate(b, "versioned.tmpl", cfg); err != nil {
			return nil, err
		}
	case "declarative":
		if err := tmpl.ExecuteTemplate(b, "declarative.tmpl", cfg); err != nil {
			return nil, err
		}
	}
	return b.Bytes(), nil
}
