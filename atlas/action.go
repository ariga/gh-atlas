package atlas

import "ariga.io/gh-atlas/atlas/gen"

type Dialect string

const (
	Postgres Dialect = "postgres"
	Mysql    Dialect = "mysql"
	Mariadb  Dialect = "maria"
	Sqlite   Dialect = "sqlite"
)

// CreateAction creates a new GitHub Action for atlas ci lint.
func CreateAction(dialect Dialect, dirPath string) ([]byte, error) {
	def := gen.Def{
		Path:    dirPath,
		Dialect: dialect,
	}
	return gen.Generate(def)
}
