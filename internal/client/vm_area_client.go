package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

type VMAreaVMListResponse struct {
	Data []VM `json:"data"`
	Meta struct {
		NextCursor *string `json:"next_cursor"`
		HasMore    bool    `json:"has_more"`
	} `json:"meta"`
}

type VMAreaAddress struct {
	Address      string  `json:"address"`
	Version      int     `json:"version"`
	Primary      bool    `json:"primary"`
	Gateway      *string `json:"gateway"`
	PrefixLength *int    `json:"prefix_length"`
	RDNS         *string `json:"rdns"`
}

type VMAreaPrivateNetwork struct {
	VPC           int64  `json:"vpc"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	PrefixLength  int    `json:"prefix_length"`
	Configuration string `json:"configuration"`
}

type VMAreaNetwork struct {
	IPv4       []VMAreaAddress        `json:"ipv4"`
	IPv6       []VMAreaAddress        `json:"ipv6"`
	IPv6Prefix *string                `json:"ipv6_prefix"`
	Private    []VMAreaPrivateNetwork `json:"private"`
	PortMbps   int                    `json:"port_mbps"`
}

type VMAreaNetworkResponse struct {
	Data VMAreaNetwork `json:"data"`
}

type VMAreaMetricPoint struct {
	Time           string   `json:"time"`
	CPU            *float64 `json:"cpu"`
	MemoryBytes    *float64 `json:"memory_bytes"`
	DiskReadBytes  *float64 `json:"disk_read_bytes"`
	DiskWriteBytes *float64 `json:"disk_write_bytes"`
	NetInBytes     *float64 `json:"net_in_bytes"`
	NetOutBytes    *float64 `json:"net_out_bytes"`
}

type VMAreaMetrics struct {
	Timeframe   string              `json:"timeframe"`
	StepSeconds int                 `json:"step_seconds"`
	Series      []VMAreaMetricPoint `json:"series"`
}
type VMAreaMetricsResponse struct {
	Data VMAreaMetrics `json:"data"`
}

type VMAreaBackup struct {
	ID         string `json:"id"`
	Type       string `json:"type"`
	CreatedAt  string `json:"created_at"`
	SizeBytes  *int64 `json:"size_bytes"`
	Restorable bool   `json:"restorable"`
}
type VMAreaBackupsResponse struct {
	Data []VMAreaBackup `json:"data"`
	Meta struct {
		AutomaticEnabled bool `json:"automatic_enabled"`
		Retention        int  `json:"retention"`
	} `json:"meta"`
}

type VMAreaAction struct {
	ID         int     `json:"id"`
	Type       string  `json:"type"`
	Status     string  `json:"status"`
	VMID       *int64  `json:"vm_id"`
	Source     string  `json:"source"`
	CreatedAt  string  `json:"created_at"`
	StartedAt  *string `json:"started_at"`
	FinishedAt *string `json:"finished_at"`
	Error      *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
type VMAreaActionListResponse struct {
	Data []VMAreaAction `json:"data"`
	Meta struct {
		NextCursor *string `json:"next_cursor"`
		HasMore    bool    `json:"has_more"`
	} `json:"meta"`
}
type VMAreaActionResponse struct {
	Data struct {
		Action VMAreaAction `json:"action"`
	} `json:"data"`
}

func (c *Client) VMAreaListVMs(ctx context.Context, status, region, billing string) ([]VM, error) {
	var all []VM
	cursor := ""
	for {
		query := url.Values{"limit": {"100"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		if status != "" {
			query.Set("status", status)
		}
		if region != "" {
			query.Set("region", region)
		}
		if billing != "" {
			query.Set("billing", billing)
		}
		var page VMAreaVMListResponse
		if err := c.networkRequest(ctx, http.MethodGet, "/vms?"+query.Encode(), nil, &page, http.StatusOK); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.Meta.HasMore {
			return all, nil
		}
		if page.Meta.NextCursor == nil || *page.Meta.NextCursor == "" {
			return nil, fmt.Errorf("VM list response reports more pages without a cursor")
		}
		cursor = *page.Meta.NextCursor
	}
}

func (c *Client) VMAreaGetNetwork(ctx context.Context, vmID int64) (*VMAreaNetworkResponse, error) {
	var result VMAreaNetworkResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("vms", vmID)+"/network", nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) VMAreaGetMetrics(ctx context.Context, vmID int64, timeframe string) (*VMAreaMetricsResponse, error) {
	query := url.Values{}
	if timeframe != "" {
		query.Set("timeframe", timeframe)
	}
	path := networkIDPath("vms", vmID) + "/metrics"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	var result VMAreaMetricsResponse
	err := c.networkRequest(ctx, http.MethodGet, path, nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) VMAreaGetBackups(ctx context.Context, vmID int64) (*VMAreaBackupsResponse, error) {
	var result VMAreaBackupsResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("vms", vmID)+"/backups", nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) VMAreaListActions(ctx context.Context, vmID *int64, status string) ([]VMAreaAction, error) {
	if vmID != nil && status != "" {
		return nil, fmt.Errorf("status filtering is only supported for account-wide actions")
	}
	path := "/actions"
	if vmID != nil {
		path = networkIDPath("vms", *vmID) + "/actions"
	}
	var all []VMAreaAction
	cursor := ""
	for {
		query := url.Values{"limit": {"100"}}
		if cursor != "" {
			query.Set("cursor", cursor)
		}
		if status != "" && vmID == nil {
			query.Set("status", status)
		}
		var page VMAreaActionListResponse
		if err := c.networkRequest(ctx, http.MethodGet, path+"?"+query.Encode(), nil, &page, http.StatusOK); err != nil {
			return nil, err
		}
		all = append(all, page.Data...)
		if !page.Meta.HasMore {
			return all, nil
		}
		if page.Meta.NextCursor == nil || *page.Meta.NextCursor == "" {
			return nil, fmt.Errorf("action list response reports more pages without a cursor")
		}
		cursor = *page.Meta.NextCursor
	}
}

func (c *Client) VMAreaAddIPv6(ctx context.Context, vmID int64, address string) (*VMAreaAddress, error) {
	var result struct {
		Data VMAreaAddress `json:"data"`
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("vms", vmID)+"/network/ipv6", map[string]string{"address": address}, &result, http.StatusCreated)
	return &result.Data, err
}

func (c *Client) VMAreaDeleteIPv6(ctx context.Context, vmID int64, address string) error {
	path := networkIDPath("vms", vmID) + "/network/ipv6/" + url.PathEscape(address)
	return c.networkRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
}

func (c *Client) VMAreaUpdateRDNS(ctx context.Context, vmID int64, address string, hostname *string) (*VMAreaAddress, error) {
	var result struct {
		Data VMAreaAddress `json:"data"`
	}
	body := map[string]any{"address": address, "hostname": hostname}
	err := c.networkRequest(ctx, http.MethodPut, networkIDPath("vms", vmID)+"/network/rdns", body, &result, http.StatusOK)
	return &result.Data, err
}

func (c *Client) VMAreaPowerAction(ctx context.Context, vmID int64, actionType string) (*VMAreaActionResponse, error) {
	var result VMAreaActionResponse
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("vms", vmID)+"/actions", map[string]string{"type": actionType}, &result, http.StatusAccepted)
	return &result, err
}

func (c *Client) VMAreaReinstall(ctx context.Context, vmID int64, image string, sshKeys []string) (*VMAreaActionResponse, error) {
	var result VMAreaActionResponse
	body := map[string]any{"type": "reinstall", "image": image}
	if sshKeys != nil {
		body["ssh_keys"] = sshKeys
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("vms", vmID)+"/actions", body, &result, http.StatusAccepted)
	return &result, err
}

func (c *Client) VMAreaRestoreBackup(ctx context.Context, vmID int64, backup string) (*VMAreaActionResponse, error) {
	var result VMAreaActionResponse
	path := networkIDPath("vms", vmID) + "/backups/" + url.PathEscape(backup) + "/restore"
	err := c.networkRequest(ctx, http.MethodPost, path, nil, &result, http.StatusAccepted)
	return &result, err
}

func (c *Client) VMAreaDeleteVMWithAction(ctx context.Context, vmID int, hostname, when string) (int, error) {
	query := url.Values{}
	query.Set("confirm", hostname)
	query.Set("when", when)
	request, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.BaseURL+"/vms/"+strconv.Itoa(vmID)+"?"+query.Encode(), nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Authorization", "Bearer "+c.Token)
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusAccepted && response.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("failed to delete VM, HTTP status code %d", response.StatusCode)
	}
	var result struct {
		Data struct {
			Action *VMAreaAction `json:"action"`
		} `json:"data"`
	}
	if response.ContentLength == 0 {
		return 0, nil
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		if err == io.EOF {
			return 0, nil
		}
		return 0, err
	}
	if result.Data.Action == nil {
		return 0, nil
	}
	return result.Data.Action.ID, nil
}
