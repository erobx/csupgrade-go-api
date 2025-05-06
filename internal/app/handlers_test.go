package app

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/erobx/csupgrade/backend/pkg/api"
)

// Should register a new user and return their data + jwt
func TestRegister(t *testing.T) {
	t.Run("registers new user", func(t *testing.T) {
		payload := api.NewUserRequest{
			Email: "test@test.com",
			Username: "testing",
			Password: "test",
		}
		b, _ := json.Marshal(payload)
		reader := bytes.NewReader(b)
		request := httptest.NewRequest(http.MethodPost, "/auth/register", reader)
		response := httptest.NewRecorder()

	}
}
