package gen

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGen(t *testing.T) {
	dir, err := os.ReadDir("testdata/")
	require.NoError(t, err)
	for _, f := range dir {
		t.Run(f.Name(), func(t *testing.T) {
			expected, err := os.ReadFile("testdata/" + f.Name())
			require.NoError(t, err)
			name := strings.TrimSuffix(f.Name(), ".yml")
			cfg := &Config{
				Path:          "migrations",
				DirName:       "name",
				DefaultBranch: "master",
				SecretName:    "ATLAS_CLOUD_TOKEN",
				Driver:        name,
				CreateDevURL:  true,
				SchemaScope:   false,
			}
			if strings.Contains(name, "atlas_config") {
				cfg.Driver = strings.Split(name, "_")[0]
				cfg.ConfigPath = "atlas.hcl"
				cfg.Env = "dev"
				cfg.CreateDevURL = false
			}
			if strings.Contains(name, "schema_scope") {
				cfg.Driver = strings.Split(name, "_")[0]
				cfg.SchemaScope = true
			}
			actual, err := Generate(cfg)
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))
		})
	}
}
