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
			Token string `json:"token"`
		}
		err := json.NewDecoder(r.Body).Decode(&input)
		require.NoError(t, err)
	}))
	client := New(srv.URL, "atlas")
	defer srv.Close()
	err := client.VerifyToken(context.Background(), "secret-token")
	require.NoError(t, err)
}
