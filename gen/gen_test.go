package gen

import (
	"fmt"
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
			cfg := &Config{
				Path:          "migrations",
				DirName:       "name",
				DefaultBranch: "master",
				SecretName:    "ATLAS_CLOUD_TOKEN",
				Driver:        strings.TrimSuffix(f.Name(), ".yml"),
			}
			actual, err := Generate(cfg)
			fmt.Print(string(actual))
			require.NoError(t, err)
			require.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(string(actual)))
		})
	}
}
