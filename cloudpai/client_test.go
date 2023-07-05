package cloudpai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_VerifyToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Variables struct {
				Token string `json:"token"`
			} `json:"variables"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		require.Equal(t, "atlas-secret-token", input.Variables.Token)
	}))
	client := New(srv.URL, "atlas-secret-token")
	defer srv.Close()
	err := client.VerifyToken(context.Background())
	require.NoError(t, err)
}
