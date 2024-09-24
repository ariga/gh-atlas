package gen

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionedFlowGen(t *testing.T) {
	dir, err := os.ReadDir("testdata/versioned")
	require.NoError(t, err)
	for _, f := range dir {
		t.Run(f.Name(), func(t *testing.T) {
			expected, err := os.ReadFile("testdata/versioned/" + f.Name())
			require.NoError(t, err)
			name := strings.TrimSuffix(f.Name(), ".yml")
			cfg := &Config{
				Flow:          "versioned",
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
			cfg.Driver = strings.ToUpper(cfg.Driver)
			actual, err := Generate(cfg)
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))
		})
	}
}

func TestDeclarativeFlowGen(t *testing.T) {
	testCases := []struct {
		name     string
		filename string
		config   *Config
	}{
		{
			name:     "full",
			filename: "plan_full.yml",
			config: &Config{
				Flow:             "declarative",
				DefaultBranch:    "master",
				Path:             "migrations",
				SecretName:       "ATLAS_CLOUD_TOKEN_X1",
				Driver:           "MYSQL",
				CreateDevURL:     true,
				SchemaScope:      false,
				From:             "atlas://myrepo:v1",
				To:               "atlas://myrepo:v2",
				ConfigPath:       "atlas.hcl",
				Env:              "GORM_MIGRATION_SOURCE",
				CloudRepo:        "myrepo",
				SetupSchemaApply: true,
			},
		},
		{
			name:     "no schema apply",
			filename: "plan_no_schema_apply.yml",
			config: &Config{
				Flow:             "declarative",
				DefaultBranch:    "master",
				Path:             "migrations",
				SecretName:       "ATLAS_CLOUD_TOKEN_X1",
				Driver:           "MYSQL",
				CreateDevURL:     true,
				SchemaScope:      false,
				From:             "atlas://myrepo:v1",
				To:               "atlas://myrepo:v2",
				ConfigPath:       "atlas.hcl",
				Env:              "GORM_MIGRATION_SOURCE",
				CloudRepo:        "myrepo",
				SetupSchemaApply: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			expected, err := os.ReadFile("testdata/declarative/" + tc.filename)
			require.NoError(t, err)
			actual, err := Generate(tc.config)
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))
		})
	}
}
