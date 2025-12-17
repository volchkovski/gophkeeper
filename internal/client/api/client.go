// Package api provides HTTP client for GophKeeper server API.
package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Client provides methods to interact with the GophKeeper server API.
type Client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

// clientConfig holds configuration for Client.
type clientConfig struct {
	timeout   time.Duration
	transport http.RoundTripper
}

// defaultConfig returns default client configuration.
func defaultConfig() *clientConfig {
	return &clientConfig{
		timeout:   30 * time.Second,
		transport: nil,
	}
}

// Option is a generic functional option type for configuring Client.
type Option[T any] func(*T)

// ClientOption is an option for configuring Client.
type ClientOption = Option[clientConfig]

// WithTimeout sets the HTTP client timeout.
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *clientConfig) {
		c.timeout = timeout
	}
}

// WithTransport sets a custom HTTP transport.
func WithTransport(transport http.RoundTripper) ClientOption {
	return func(c *clientConfig) {
		c.transport = transport
	}
}

// NewClient creates a new API client with functional options.
func NewClient(baseURL string, opts ...ClientOption) *Client {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	httpClient := &http.Client{
		Timeout:   cfg.timeout,
		Transport: cfg.transport,
	}

	return &Client{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// SetToken sets the authentication token.
func (c *Client) SetToken(token string) {
	c.token = token
}

// Request types

// RegisterRequest represents a registration request.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RegisterResponse represents a registration response.
type RegisterResponse struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// LoginRequest represents a login request.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse represents a login response.
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshRequest represents a token refresh request.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// SecretRequest represents a create/update secret request.
type SecretRequest struct {
	Type          string `json:"type"`
	Name          string `json:"name"`
	EncryptedData []byte `json:"encrypted_data"`
	Metadata      string `json:"metadata,omitempty"`
}

// SecretResponse represents a secret in API responses.
type SecretResponse struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	EncryptedData []byte `json:"encrypted_data"`
	Metadata      string `json:"metadata,omitempty"`
	Version       int64  `json:"version"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

// SecretsListResponse represents a list of secrets.
type SecretsListResponse struct {
	Secrets []SecretResponse `json:"secrets"`
}

// SyncSecretRequest represents a secret in sync request.
type SyncSecretRequest struct {
	ID            string `json:"id"`
	Type          string `json:"type"`
	Name          string `json:"name"`
	EncryptedData []byte `json:"encrypted_data"`
	Metadata      string `json:"metadata,omitempty"`
	Version       int64  `json:"version"`
	UpdatedAt     string `json:"updated_at"`
}

// SyncRequest represents a sync request.
type SyncRequest struct {
	Secrets []SyncSecretRequest `json:"secrets"`
}

// SyncResponse represents a sync response.
type SyncResponse struct {
	UpdatedSecrets []SecretResponse    `json:"updated_secrets"`
	Conflicts      []ConflictResponse  `json:"conflicts,omitempty"`
}

// ConflictResponse represents a sync conflict.
type ConflictResponse struct {
	ClientVersion SecretResponse `json:"client_version"`
	ServerVersion SecretResponse `json:"server_version"`
}

// HealthResponse represents a health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// VersionResponse represents a version response.
type VersionResponse struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Error string `json:"error"`
}

// API methods

// Health checks server health.
func (c *Client) Health() (*HealthResponse, error) {
	resp, err := c.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var health HealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &health, nil
}

// Version returns server version.
func (c *Client) Version() (*VersionResponse, error) {
	resp, err := c.doRequest("GET", "/api/v1/version", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var version VersionResponse
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &version, nil
}

// Register registers a new user.
func (c *Client) Register(username, password string) (*RegisterResponse, error) {
	req := RegisterRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/register", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// Login authenticates a user.
func (c *Client) Login(username, password string) (*LoginResponse, error) {
	req := LoginRequest{
		Username: username,
		Password: password,
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/login", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.token = result.AccessToken
	return &result, nil
}

// RefreshToken refreshes the access token.
func (c *Client) RefreshToken(refreshToken string) (*LoginResponse, error) {
	req := RefreshRequest{
		RefreshToken: refreshToken,
	}

	resp, err := c.doRequest("POST", "/api/v1/auth/refresh", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result LoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.token = result.AccessToken
	return &result, nil
}

// CreateSecret creates a new secret.
func (c *Client) CreateSecret(secretType, name string, encryptedData []byte, metadata string) (*SecretResponse, error) {
	req := SecretRequest{
		Type:          secretType,
		Name:          name,
		EncryptedData: encryptedData,
		Metadata:      metadata,
	}

	resp, err := c.doRequest("POST", "/api/v1/secrets", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseError(resp)
	}

	var result SecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetSecret retrieves a secret by ID.
func (c *Client) GetSecret(id uuid.UUID) (*SecretResponse, error) {
	resp, err := c.doRequest("GET", fmt.Sprintf("/api/v1/secrets/%s", id), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result SecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// ListSecrets retrieves all secrets.
func (c *Client) ListSecrets() (*SecretsListResponse, error) {
	resp, err := c.doRequest("GET", "/api/v1/secrets", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result SecretsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateSecret updates a secret.
func (c *Client) UpdateSecret(id uuid.UUID, secretType, name string, encryptedData []byte, metadata string) (*SecretResponse, error) {
	req := SecretRequest{
		Type:          secretType,
		Name:          name,
		EncryptedData: encryptedData,
		Metadata:      metadata,
	}

	resp, err := c.doRequest("PUT", fmt.Sprintf("/api/v1/secrets/%s", id), req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result SecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// DeleteSecret deletes a secret.
func (c *Client) DeleteSecret(id uuid.UUID) error {
	resp, err := c.doRequest("DELETE", fmt.Sprintf("/api/v1/secrets/%s", id), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseError(resp)
	}

	return nil
}

// Sync synchronizes secrets with the server.
func (c *Client) Sync(secrets []SyncSecretRequest) (*SyncResponse, error) {
	req := SyncRequest{
		Secrets: secrets,
	}

	resp, err := c.doRequest("POST", "/api/v1/secrets/sync", req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	var result SyncResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// doRequest performs an HTTP request.
func (c *Client) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	return c.httpClient.Do(req)
}

// parseError parses an error response.
func (c *Client) parseError(resp *http.Response) error {
	body, _ := io.ReadAll(resp.Body)
	
	var errResp ErrorResponse
	if err := json.Unmarshal(body, &errResp); err == nil && errResp.Error != "" {
		return fmt.Errorf("%s", errResp.Error)
	}

	return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
}

