package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

type AccessRoleRef struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

type AccessPolicy struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Provided    bool            `json:"provided"`
	Document    json.RawMessage `json:"document"`
	RolesCount  *int64          `json:"roles_count"`
	CreatedAt   *string         `json:"created_at"`
}

type AccessRole struct {
	ID          int64   `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	PolicyIDs   []int64 `json:"policy_ids"`
	CreatedAt   *string `json:"created_at"`
}

type AccessAPIKey struct {
	ID             int64           `json:"id"`
	Name           string          `json:"name"`
	Prefix         *string         `json:"prefix"`
	Enabled        bool            `json:"enabled"`
	Role           *AccessRoleRef  `json:"role"`
	ExchangeOnly   bool            `json:"exchange_only"`
	AssumableRoles []AccessRoleRef `json:"assumable_roles"`
	IPAddresses    []string        `json:"ip_addresses"`
	ExpiresAt      *string         `json:"expires_at"`
	LastUsedAt     *string         `json:"last_used_at"`
	CreatedAt      *string         `json:"created_at"`
	// Token is only present in the creation response and is never returned again.
	Token string `json:"token"`
}

type AccessOIDCProvider struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Issuer   string `json:"issuer"`
	Audience string `json:"audience"`
	Enabled  bool   `json:"enabled"`
}

type AccessClaimCondition struct {
	Claim string `json:"claim"`
	Value string `json:"value"`
}

type AccessOIDCBinding struct {
	ID              int64                  `json:"id"`
	ProviderID      int64                  `json:"provider_id"`
	Name            string                 `json:"name"`
	ClaimConditions []AccessClaimCondition `json:"claim_conditions"`
	Role            *AccessRoleRef         `json:"role"`
	ExchangeOnly    bool                   `json:"exchange_only"`
	AssumableRoles  []AccessRoleRef        `json:"assumable_roles"`
	Enabled         bool                   `json:"enabled"`
	ExpiresAt       *string                `json:"expires_at"`
	CreatedAt       *string                `json:"created_at"`
}

func accessDo[T any](ctx context.Context, c *Client, method, path string, body any, status int, idempotencyKey string) (T, error) {
	var envelope struct {
		Data T `json:"data"`
	}
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"Idempotency-Key": idempotencyKey}
	}
	err := c.networkRequestWithHeaders(ctx, method, path, body, &envelope, []int{status}, headers)
	return envelope.Data, err
}

func accessPath(collection string, id int64) string {
	return "/" + collection + "/" + strconv.FormatInt(id, 10)
}

func (c *Client) AccessListPolicies(ctx context.Context) ([]AccessPolicy, error) {
	return accessDo[[]AccessPolicy](ctx, c, http.MethodGet, "/policies", nil, http.StatusOK, "")
}

func (c *Client) AccessGetPolicy(ctx context.Context, id int64) (AccessPolicy, error) {
	return accessDo[AccessPolicy](ctx, c, http.MethodGet, accessPath("policies", id), nil, http.StatusOK, "")
}

func (c *Client) AccessCreatePolicy(ctx context.Context, body map[string]any, idempotencyKey string) (AccessPolicy, error) {
	return accessDo[AccessPolicy](ctx, c, http.MethodPost, "/policies", body, http.StatusCreated, idempotencyKey)
}

func (c *Client) AccessUpdatePolicy(ctx context.Context, id int64, body map[string]any, idempotencyKey string) (AccessPolicy, error) {
	return accessDo[AccessPolicy](ctx, c, http.MethodPut, accessPath("policies", id), body, http.StatusOK, idempotencyKey)
}

func (c *Client) AccessDeletePolicy(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, accessPath("policies", id), nil, nil, http.StatusNoContent)
}

func (c *Client) AccessListRoles(ctx context.Context) ([]AccessRole, error) {
	return accessDo[[]AccessRole](ctx, c, http.MethodGet, "/roles", nil, http.StatusOK, "")
}

func (c *Client) AccessGetRole(ctx context.Context, id int64) (AccessRole, error) {
	return accessDo[AccessRole](ctx, c, http.MethodGet, accessPath("roles", id), nil, http.StatusOK, "")
}

func (c *Client) AccessCreateRole(ctx context.Context, body map[string]any, idempotencyKey string) (AccessRole, error) {
	return accessDo[AccessRole](ctx, c, http.MethodPost, "/roles", body, http.StatusCreated, idempotencyKey)
}

func (c *Client) AccessUpdateRole(ctx context.Context, id int64, body map[string]any, idempotencyKey string) (AccessRole, error) {
	return accessDo[AccessRole](ctx, c, http.MethodPut, accessPath("roles", id), body, http.StatusOK, idempotencyKey)
}

func (c *Client) AccessDeleteRole(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, accessPath("roles", id), nil, nil, http.StatusNoContent)
}

func (c *Client) AccessListAPIKeys(ctx context.Context) ([]AccessAPIKey, error) {
	return accessDo[[]AccessAPIKey](ctx, c, http.MethodGet, "/api-keys", nil, http.StatusOK, "")
}

func (c *Client) AccessGetAPIKey(ctx context.Context, id int64) (AccessAPIKey, error) {
	return accessDo[AccessAPIKey](ctx, c, http.MethodGet, accessPath("api-keys", id), nil, http.StatusOK, "")
}

// AccessCreateAPIKey sends no Idempotency-Key: the API does not replay key creation because the secret is shown once.
func (c *Client) AccessCreateAPIKey(ctx context.Context, body map[string]any) (AccessAPIKey, error) {
	return accessDo[AccessAPIKey](ctx, c, http.MethodPost, "/api-keys", body, http.StatusCreated, "")
}

func (c *Client) AccessDeleteAPIKey(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, accessPath("api-keys", id), nil, nil, http.StatusNoContent)
}

func (c *Client) AccessListOIDCProviders(ctx context.Context) ([]AccessOIDCProvider, error) {
	return accessDo[[]AccessOIDCProvider](ctx, c, http.MethodGet, "/oidc/providers", nil, http.StatusOK, "")
}

func (c *Client) AccessGetOIDCBinding(ctx context.Context, id int64) (AccessOIDCBinding, error) {
	return accessDo[AccessOIDCBinding](ctx, c, http.MethodGet, "/oidc/bindings/"+strconv.FormatInt(id, 10), nil, http.StatusOK, "")
}

func (c *Client) AccessCreateOIDCBinding(ctx context.Context, body map[string]any, idempotencyKey string) (AccessOIDCBinding, error) {
	return accessDo[AccessOIDCBinding](ctx, c, http.MethodPost, "/oidc/bindings", body, http.StatusCreated, idempotencyKey)
}

func (c *Client) AccessUpdateOIDCBinding(ctx context.Context, id int64, body map[string]any, idempotencyKey string) (AccessOIDCBinding, error) {
	return accessDo[AccessOIDCBinding](ctx, c, http.MethodPut, "/oidc/bindings/"+strconv.FormatInt(id, 10), body, http.StatusOK, idempotencyKey)
}

func (c *Client) AccessDeleteOIDCBinding(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, "/oidc/bindings/"+strconv.FormatInt(id, 10), nil, nil, http.StatusNoContent)
}
