package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var ErrVMNotFound = errors.New("VM not found")

type Client struct {
	BaseURL    string
	Token      string
	HTTPClient *http.Client
}

func NewClient(baseURL, token string) *Client {
	if baseURL == "" {
		baseURL = "https://api.ffxf.net/v1"
	}
	return &Client{
		BaseURL:    baseURL,
		Token:      token,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type CreateVmRequest struct {
	Plan     string   `json:"plan"`
	Region   string   `json:"region"`
	Image    string   `json:"image"`
	Hostname string   `json:"hostname"`
	Billing  string   `json:"billing"`
	SSHKeys  []string `json:"ssh_keys,omitempty"`
}

type CreateVmResponse struct {
	Data struct {
		VM struct {
			ID       int    `json:"id"`
			Hostname string `json:"hostname"`
			Status   string `json:"status"`
			IPv4     string `json:"ipv4"`
			IPv6     string `json:"ipv6"`
		} `json:"vm"`
		Action struct {
			ID int `json:"id"`
		} `json:"action"`
	} `json:"data"`
}

type VM struct {
	ID       int     `json:"id"`
	Hostname string  `json:"hostname"`
	Status   string  `json:"status"`
	Plan     string  `json:"plan"`
	Region   string  `json:"region"`
	Image    string  `json:"image"`
	IPv4     *string `json:"ipv4"`
	Billing  struct {
		Mode string `json:"mode"`
	} `json:"billing"`
}

type VMResponse struct {
	Data VM `json:"data"`
}

type UpdateVmRequest struct {
	Hostname string `json:"hostname"`
}

type ActionResponse struct {
	Data struct {
		ID     int    `json:"id"`
		Status string `json:"status"` // "running", "completed", "error"
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	} `json:"data"`
}

func (c *Client) CreateVM(ctx context.Context, req CreateVmRequest, idempotencyKey string) (*CreateVmResponse, error) {
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/vms", bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Idempotency-Key", idempotencyKey)

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP error %d during creation", res.StatusCode)
	}

	var result CreateVmResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) GetVM(ctx context.Context, vmID int64) (*VMResponse, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/vms/%d", c.BaseURL, vmID), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, ErrVMNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP error %d during read", res.StatusCode)
	}

	var result VMResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) UpdateVM(ctx context.Context, vmID int64, hostname string) (*VMResponse, error) {
	body, err := json.Marshal(UpdateVmRequest{Hostname: hostname})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPatch, fmt.Sprintf("%s/vms/%d", c.BaseURL, vmID), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)
	httpReq.Header.Set("Content-Type", "application/json")

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP error %d during update", res.StatusCode)
	}

	var result VMResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) WaitForAction(ctx context.Context, actionID int) error {
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			httpReq, _ := http.NewRequestWithContext(ctx, "GET", fmt.Sprintf("%s/actions/%d", c.BaseURL, actionID), nil)
			httpReq.Header.Set("Authorization", "Bearer "+c.Token)

			res, err := c.HTTPClient.Do(httpReq)
			if err != nil {
				return err
			}

			var actRes ActionResponse
			json.NewDecoder(res.Body).Decode(&actRes)
			res.Body.Close()

			switch actRes.Data.Status {
			case "completed":
				return nil
			case "error":
				msg := "unknown error"
				if actRes.Data.Error != nil {
					msg = actRes.Data.Error.Message
				}
				return fmt.Errorf("the action failed: %s", msg)
			}
		}
	}
}

func (c *Client) DeleteVM(ctx context.Context, vmID int, hostname string) error {
	url := fmt.Sprintf("%s/vms/%d?confirm=%s&when=now", c.BaseURL, vmID, hostname)
	httpReq, _ := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusAccepted && res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to delete VM, HTTP status code %d", res.StatusCode)
	}

	return nil
}
