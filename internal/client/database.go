package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

type Database struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	Status         string            `json:"status"`
	Engine         string            `json:"engine"`
	Version        string            `json:"version"`
	EngineVersion  *string           `json:"engine_version"`
	Region         *string           `json:"region"`
	VPC            int64             `json:"vpc"`
	SizeGB         *int64            `json:"size_gb"`
	EvictionPolicy *string           `json:"eviction_policy"`
	Maintenance    json.RawMessage   `json:"maintenance"`
	Billing        json.RawMessage   `json:"billing"`
	Connection     json.RawMessage   `json:"connection"`
	PublicAccess   json.RawMessage   `json:"public_access"`
	Disk           json.RawMessage   `json:"disk"`
	Users          []DatabaseUser    `json:"users"`
	Databases      []LogicalDatabase `json:"databases"`
	CreatedAt      string            `json:"created_at"`
	DatabaseJSON   json.RawMessage   `json:"-"`
}

func (database *Database) UnmarshalJSON(data []byte) error {
	type databaseAlias Database
	if err := json.Unmarshal(data, (*databaseAlias)(database)); err != nil {
		return err
	}
	database.DatabaseJSON = append(database.DatabaseJSON[:0], data...)
	return nil
}

type DatabaseUser struct {
	Username  string   `json:"username"`
	Admin     bool     `json:"admin"`
	Databases []string `json:"databases"`
	ReadOnly  bool     `json:"read_only"`
}

type LogicalDatabase struct {
	Name       string   `json:"name"`
	Extensions []string `json:"extensions"`
}

type DatabaseResponse struct {
	Data Database `json:"data"`
}

type DatabaseListResponse struct {
	Data  []Database `json:"data"`
	Quota int        `json:"quota"`
}

func (c *Client) DatabaseCall(ctx context.Context, method, path string, body, target any, expected []int, headers map[string]string) error {
	return c.networkRequestWithHeaders(ctx, method, path, body, target, expected, headers)
}

func (c *Client) DatabaseList(ctx context.Context) (*DatabaseListResponse, error) {
	var result DatabaseListResponse
	err := c.networkRequest(ctx, http.MethodGet, "/databases", nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) DatabaseGet(ctx context.Context, id int64) (*DatabaseResponse, error) {
	var result DatabaseResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("databases", id), nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) DatabaseCreate(ctx context.Context, body map[string]any, key string) (*DatabaseResponse, error) {
	var result DatabaseResponse
	err := c.networkRequestWithHeaders(ctx, http.MethodPost, "/databases", body, &result, []int{http.StatusAccepted}, map[string]string{"Idempotency-Key": key})
	return &result, err
}

func (c *Client) DatabasePatch(ctx context.Context, id int64, body map[string]any) (*DatabaseResponse, error) {
	var result DatabaseResponse
	err := c.networkRequest(ctx, http.MethodPatch, networkIDPath("databases", id), body, &result, http.StatusOK)
	return &result, err
}

func (c *Client) DatabaseResize(ctx context.Context, id, size int64, key string) (*DatabaseResponse, error) {
	var result DatabaseResponse
	err := c.networkRequestWithHeaders(ctx, http.MethodPost, networkIDPath("databases", id)+"/resize", map[string]int64{"size_gb": size}, &result, []int{http.StatusAccepted}, map[string]string{"Idempotency-Key": key})
	return &result, err
}

func (c *Client) DatabaseDelete(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, networkIDPath("databases", id), nil, nil, http.StatusAccepted, http.StatusNoContent)
}

func (c *Client) DatabasePath(id int64, suffix string) string {
	path := networkIDPath("databases", id)
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}

func (c *Client) DatabaseOptions(ctx context.Context, target any) error {
	return c.networkRequest(ctx, http.MethodGet, "/databases/options", nil, target, http.StatusOK)
}

func (c *Client) DatabaseEndpoint(ctx context.Context, id int64, suffix string, query url.Values, target any) error {
	path := c.DatabasePath(id, suffix)
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return c.networkRequest(ctx, http.MethodGet, path, nil, target, http.StatusOK)
}

func (c *Client) DatabaseUserPath(id int64, username, suffix string) string {
	path := c.DatabasePath(id, "users") + "/" + url.PathEscape(username)
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}

func (c *Client) DatabaseLogicalPath(id int64, name, suffix string) string {
	path := c.DatabasePath(id, "databases") + "/" + url.PathEscape(name)
	if suffix != "" {
		path += "/" + suffix
	}
	return path
}

func (c *Client) DatabaseDerivedCreate(ctx context.Context, source int64, operation string, body map[string]any, key string) (*DatabaseResponse, error) {
	var result DatabaseResponse
	path := c.DatabasePath(source, operation)
	err := c.networkRequestWithHeaders(ctx, http.MethodPost, path, body, &result, []int{http.StatusAccepted}, map[string]string{"Idempotency-Key": key})
	return &result, err
}
