package cloudapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestClient_ValidateToken(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Variables struct {
				Token string `json:"token"`
			} `json:"variables"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
		require.Equal(t, "atlas-secret-token", input.Variables.Token)
		body, err := json.Marshal(input)
		require.NoError(t, err)
		_, err = w.Write(body)
		require.NoError(t, err)
	}))
	client := New(srv.URL, "atlas-secret-token")
	defer srv.Close()
	err := client.ValidateToken(context.Background())
	require.NoError(t, err)
}
