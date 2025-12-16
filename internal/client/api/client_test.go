package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	client := NewClient("http://localhost:8080", 30*time.Second)

	require.NotNil(t, client)
	assert.Equal(t, "http://localhost:8080", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.Empty(t, client.token)
}

func TestClient_SetToken(t *testing.T) {
	client := NewClient("http://localhost:8080", 30*time.Second)
	client.SetToken("test-token")

	assert.Equal(t, "test-token", client.token)
}

func TestClient_Health(t *testing.T) {
	t.Run("successful health check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/health", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		health, err := client.Health()

		require.NoError(t, err)
		assert.Equal(t, "ok", health.Status)
	})

	t.Run("server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(HealthResponse{Status: "error"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		health, err := client.Health()

		require.NoError(t, err) // Health still decodes but may have error status
		assert.NotNil(t, health)
		assert.Equal(t, "error", health.Status)
	})
}

func TestClient_Version(t *testing.T) {
	t.Run("successful version check", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/version", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(VersionResponse{
				Version:   "1.0.0",
				BuildDate: "2024-01-01",
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		version, err := client.Version()

		require.NoError(t, err)
		assert.Equal(t, "1.0.0", version.Version)
		assert.Equal(t, "2024-01-01", version.BuildDate)
	})
}

func TestClient_Register(t *testing.T) {
	t.Run("successful registration", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/auth/register", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req RegisterRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "testuser", req.Username)
			assert.Equal(t, "password123", req.Password)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(RegisterResponse{
				UserID:   "user-123",
				Username: "testuser",
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.Register("testuser", "password123")

		require.NoError(t, err)
		assert.Equal(t, "user-123", resp.UserID)
		assert.Equal(t, "testuser", resp.Username)
	})

	t.Run("registration failure - user exists", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "user already exists"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.Register("existinguser", "password123")

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "user already exists")
	})
}

func TestClient_Login(t *testing.T) {
	t.Run("successful login", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/auth/login", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req LoginRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "testuser", req.Username)
			assert.Equal(t, "password123", req.Password)

			json.NewEncoder(w).Encode(LoginResponse{
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-123",
				ExpiresIn:    3600,
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.Login("testuser", "password123")

		require.NoError(t, err)
		assert.Equal(t, "access-token-123", resp.AccessToken)
		assert.Equal(t, "refresh-token-123", resp.RefreshToken)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
		// Token should be set on client
		assert.Equal(t, "access-token-123", client.token)
	})

	t.Run("login failure - invalid credentials", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid credentials"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.Login("testuser", "wrongpassword")

		require.Error(t, err)
		assert.Nil(t, resp)
		assert.Contains(t, err.Error(), "invalid credentials")
	})
}

func TestClient_RefreshToken(t *testing.T) {
	t.Run("successful refresh", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/auth/refresh", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			var req RefreshRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "old-refresh-token", req.RefreshToken)

			json.NewEncoder(w).Encode(LoginResponse{
				AccessToken:  "new-access-token",
				RefreshToken: "new-refresh-token",
				ExpiresIn:    3600,
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.RefreshToken("old-refresh-token")

		require.NoError(t, err)
		assert.Equal(t, "new-access-token", resp.AccessToken)
		assert.Equal(t, "new-access-token", client.token)
	})

	t.Run("refresh failure - invalid token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "invalid token"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.RefreshToken("invalid-token")

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_CreateSecret(t *testing.T) {
	t.Run("successful creation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets", r.URL.Path)
			assert.Equal(t, "POST", r.Method)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			var req SecretRequest
			json.NewDecoder(r.Body).Decode(&req)
			assert.Equal(t, "login_password", req.Type)
			assert.Equal(t, "my-secret", req.Name)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(SecretResponse{
				ID:            "secret-123",
				Type:          "login_password",
				Name:          "my-secret",
				EncryptedData: []byte("encrypted"),
				Version:       1,
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.CreateSecret("login_password", "my-secret", []byte("encrypted"), "metadata")

		require.NoError(t, err)
		assert.Equal(t, "secret-123", resp.ID)
		assert.Equal(t, "my-secret", resp.Name)
	})

	t.Run("creation failure - conflict", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "secret already exists"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.CreateSecret("login_password", "existing", []byte("encrypted"), "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_GetSecret(t *testing.T) {
	t.Run("successful get", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			json.NewEncoder(w).Encode(SecretResponse{
				ID:            secretID.String(),
				Type:          "text",
				Name:          "my-secret",
				EncryptedData: []byte("encrypted"),
				Version:       1,
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.GetSecret(secretID)

		require.NoError(t, err)
		assert.Equal(t, secretID.String(), resp.ID)
	})

	t.Run("get failure - not found", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "secret not found"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.GetSecret(secretID)

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_ListSecrets(t *testing.T) {
	t.Run("successful list", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets", r.URL.Path)
			assert.Equal(t, "GET", r.Method)

			json.NewEncoder(w).Encode(SecretsListResponse{
				Secrets: []SecretResponse{
					{ID: "1", Name: "secret1"},
					{ID: "2", Name: "secret2"},
				},
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.ListSecrets()

		require.NoError(t, err)
		assert.Len(t, resp.Secrets, 2)
	})

	t.Run("list failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "server error"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.ListSecrets()

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_UpdateSecret(t *testing.T) {
	t.Run("successful update", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
			assert.Equal(t, "PUT", r.Method)

			json.NewEncoder(w).Encode(SecretResponse{
				ID:      secretID.String(),
				Name:    "updated-secret",
				Version: 2,
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.UpdateSecret(secretID, "text", "updated-secret", []byte("new-data"), "")

		require.NoError(t, err)
		assert.Equal(t, "updated-secret", resp.Name)
		assert.Equal(t, int64(2), resp.Version)
	})

	t.Run("update failure - not found", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "not found"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.UpdateSecret(secretID, "text", "secret", []byte("data"), "")

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_DeleteSecret(t *testing.T) {
	t.Run("successful delete", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets/"+secretID.String(), r.URL.Path)
			assert.Equal(t, "DELETE", r.Method)

			w.WriteHeader(http.StatusNoContent)
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		err := client.DeleteSecret(secretID)

		require.NoError(t, err)
	})

	t.Run("delete failure - not found", func(t *testing.T) {
		secretID := uuid.New()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "not found"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		err := client.DeleteSecret(secretID)

		require.Error(t, err)
	})
}

func TestClient_Sync(t *testing.T) {
	t.Run("successful sync", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/api/v1/secrets/sync", r.URL.Path)
			assert.Equal(t, "POST", r.Method)

			json.NewEncoder(w).Encode(SyncResponse{
				UpdatedSecrets: []SecretResponse{
					{ID: "1", Name: "server-secret"},
				},
			})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.Sync([]SyncSecretRequest{
			{ID: "2", Name: "client-secret", Version: 1},
		})

		require.NoError(t, err)
		assert.Len(t, resp.UpdatedSecrets, 1)
	})

	t.Run("sync failure", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "sync failed"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("test-token")

		resp, err := client.Sync([]SyncSecretRequest{})

		require.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestClient_doRequest(t *testing.T) {
	t.Run("request without body", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, err := client.doRequest("GET", "/test", nil)

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
		resp.Body.Close()
	})

	t.Run("request with authorization header", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		client.SetToken("my-token")
		resp, err := client.doRequest("GET", "/test", nil)

		require.NoError(t, err)
		resp.Body.Close()
	})
}

func TestClient_parseError(t *testing.T) {
	t.Run("parse error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{Error: "bad request"})
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, _ := client.doRequest("GET", "/test", nil)
		err := client.parseError(resp)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "bad request")
	})

	t.Run("parse non-json error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("plain text error"))
		}))
		defer server.Close()

		client := NewClient(server.URL, 5*time.Second)
		resp, _ := client.doRequest("GET", "/test", nil)
		err := client.parseError(resp)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 500")
	})
}

