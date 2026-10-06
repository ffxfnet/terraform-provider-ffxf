package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

var ErrVMNotFound = errors.New("VM not found")
var ErrVPCNotFound = errors.New("VPC not found")

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

type CreateVPCRequest struct {
	Name   string `json:"name"`
	CIDR   string `json:"cidr"`
	Region string `json:"region"`
}

type VPC struct {
	ID              int    `json:"id"`
	Name            string `json:"name"`
	CIDR            string `json:"cidr"`
	Region          string `json:"region"`
	Status          string `json:"status"`
	Gateway         string `json:"gateway"`
	InternetGateway bool   `json:"internet_gateway"`
	CreatedAt       string `json:"created_at"`
}

type VPCResponse struct {
	Data VPC `json:"data"`
}

type Region struct {
	Slug    string `json:"slug"`
	Name    string `json:"name"`
	Country string `json:"country"`
	Status  string `json:"status"`
	IPv4    bool   `json:"ipv4"`
	IPv6    bool   `json:"ipv6"`
}

type RegionsResponse struct {
	Data []Region `json:"data"`
}

type Plan struct {
	Slug        string   `json:"slug"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	VCPU        int      `json:"vcpu"`
	MemoryMB    int      `json:"memory_mb"`
	DiskGB      int      `json:"disk_gb"`
	TrafficTB   *float64 `json:"traffic_tb"`
	PortMbps    int      `json:"port_mbps"`
	Regions     []string `json:"regions"`
	Status      string   `json:"status"`
	Prices      []Price  `json:"prices"`
}

type Price struct {
	Currency              string  `json:"currency"`
	Hourly                string  `json:"hourly"`
	HourlyMonthEquivalent string  `json:"hourly_month_equivalent"`
	HourlyStopped         *string `json:"hourly_stopped"`
	Monthly               string  `json:"monthly"`
	Annual                string  `json:"annual"`
	SetupFee              string  `json:"setup_fee"`
}

type PlanResponse struct {
	Data Plan `json:"data"`
}

type Image struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	Family          string   `json:"family"`
	Category        string   `json:"category"`
	Version         *string  `json:"version"`
	Status          string   `json:"status"`
	Regions         []string `json:"regions"`
	MinDiskGB       *int     `json:"min_disk_gb"`
	MinMemoryMB     *int     `json:"min_memory_mb"`
	DefaultUser     string   `json:"default_user"`
	SupportsSSHKeys bool     `json:"supports_ssh_keys"`
}

type ImageResponse struct {
	Data Image `json:"data"`
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

func (c *Client) ListRegions(ctx context.Context) (*RegionsResponse, error) {
	var result RegionsResponse
	if err := c.get(ctx, "/regions", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetPlan(ctx context.Context, slug string) (*PlanResponse, error) {
	var result PlanResponse
	if err := c.get(ctx, "/plans/"+url.PathEscape(slug), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetImage(ctx context.Context, slug string) (*ImageResponse, error) {
	var result ImageResponse
	if err := c.get(ctx, "/images/"+url.PathEscape(slug), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) CreateVPC(ctx context.Context, req CreateVPCRequest) (*VPCResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/vpcs", bytes.NewBuffer(body))
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
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("API HTTP error %d during VPC creation", res.StatusCode)
	}

	var result VPCResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetVPC(ctx context.Context, vpcID int64) (*VPCResponse, error) {
	path := fmt.Sprintf("%s/vpcs/%d", c.BaseURL, vpcID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
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
		return nil, ErrVPCNotFound
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API HTTP error %d during VPC read", res.StatusCode)
	}

	var result VPCResponse
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteVPC(ctx context.Context, vpcID int64) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, fmt.Sprintf("%s/vpcs/%d", c.BaseURL, vpcID), nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNoContent && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("API HTTP error %d during VPC deletion", res.StatusCode)
	}
	return nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+path, nil)
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.Token)

	res, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("API HTTP error %d during GET %s", res.StatusCode, path)
	}
	if err := json.NewDecoder(res.Body).Decode(target); err != nil {
		return err
	}
	return nil
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
