package main

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIntegration(t *testing.T) {
	if os.Getenv("GH_ATLAS_INTEGRATION") != "1" {
		t.Skipf("skipping integration test, run with env GH_ATLAS_INTEGRATION=1")
	}
	err := exec.Command("gh", "atlas", "init-ci", "-t", os.Getenv("ATLAS_CLOUD_TOKEN"), "--driver=mysql", "testdata/migrations").Run()
	require.NoError(t, err)
}
