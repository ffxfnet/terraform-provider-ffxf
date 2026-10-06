package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var ErrNetworkResourceNotFound = errors.New("network resource not found")

type FirewallRule struct {
	Direction    string   `json:"direction,omitempty"`
	Protocol     string   `json:"protocol"`
	Ports        *string  `json:"ports,omitempty"`
	Sources      []string `json:"sources,omitempty"`
	Destinations []string `json:"destinations,omitempty"`
	Description  *string  `json:"description,omitempty"`
}

type FirewallMember struct {
	VM       int64  `json:"vm"`
	Hostname string `json:"hostname"`
	Status   string `json:"status"`
}

type Firewall struct {
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	Rules     []FirewallRule   `json:"rules"`
	Members   []FirewallMember `json:"members"`
	CreatedAt string           `json:"created_at"`
}

type FirewallResponse struct {
	Data            Firewall `json:"data"`
	PendingMachines int      `json:"pending_machines,omitempty"`
}

type FirewallListResponse struct {
	Data  []Firewall `json:"data"`
	Quota int        `json:"quota"`
}

type LoadBalancerTarget struct {
	ID         int64  `json:"id,omitempty"`
	VM         int64  `json:"vm"`
	Hostname   string `json:"hostname,omitempty"`
	Address    string `json:"address,omitempty"`
	Port       int    `json:"port"`
	Weight     int    `json:"weight"`
	Drain      bool   `json:"drain"`
	Health     string `json:"health,omitempty"`
	Sessions   int64  `json:"sessions,omitempty"`
	ResponseMS *int   `json:"response_ms,omitempty"`
}

type LoadBalancerPool struct {
	ID             int64                `json:"id"`
	Name           string               `json:"name"`
	Protocol       string               `json:"protocol"`
	Algorithm      string               `json:"algorithm"`
	HealthType     string               `json:"health_type"`
	HealthPath     *string              `json:"health_path"`
	HealthInterval int                  `json:"health_interval"`
	SendProxy      bool                 `json:"send_proxy"`
	Targets        []LoadBalancerTarget `json:"targets"`
}

type LoadBalancerRule struct {
	Hostname   *string `json:"hostname,omitempty"`
	PathPrefix *string `json:"path_prefix,omitempty"`
	Pool       int64   `json:"pool"`
}

type LoadBalancerListener struct {
	ID            int64              `json:"id"`
	Port          int                `json:"port"`
	Protocol      string             `json:"protocol"`
	DefaultPool   *int64             `json:"default_pool"`
	RedirectHTTPS bool               `json:"redirect_https"`
	Sources       []string           `json:"sources"`
	Rules         []LoadBalancerRule `json:"rules"`
}

type LoadBalancerCertificate struct {
	Hostname  string  `json:"hostname"`
	Status    string  `json:"status"`
	ExpiresAt *string `json:"expires_at"`
	Error     *string `json:"error"`
}

type LoadBalancerBilling struct {
	Mode     string `json:"mode"`
	Rate     string `json:"rate"`
	Currency string `json:"currency"`
}

type LoadBalancer struct {
	ID            int64                     `json:"id"`
	Name          string                    `json:"name"`
	Status        string                    `json:"status"`
	VPC           int64                     `json:"vpc"`
	IPv4          *string                   `json:"ipv4"`
	IPv6          *string                   `json:"ipv6"`
	DNSName       *string                   `json:"dns_name"`
	ConfigApplied bool                      `json:"config_applied"`
	LastSeenAt    *string                   `json:"last_seen_at"`
	LastError     *string                   `json:"last_error"`
	Pools         []LoadBalancerPool        `json:"pools"`
	Listeners     []LoadBalancerListener    `json:"listeners"`
	Certificates  []LoadBalancerCertificate `json:"certificates"`
	Billing       *LoadBalancerBilling      `json:"billing"`
	CreatedAt     string                    `json:"created_at"`
}

type LoadBalancerResponse struct {
	Data LoadBalancer `json:"data"`
}

type LoadBalancerListResponse struct {
	Data  []LoadBalancer `json:"data"`
	Quota int            `json:"quota"`
}

type LoadBalancerMetricPoint struct {
	At          string `json:"at"`
	Requests    int64  `json:"requests"`
	HTTP2xx     int64  `json:"http_2xx"`
	HTTP3xx     int64  `json:"http_3xx"`
	HTTP4xx     int64  `json:"http_4xx"`
	HTTP5xx     int64  `json:"http_5xx"`
	Connections int64  `json:"connections"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
	SessionsMax int64  `json:"sessions_max"`
}

type LoadBalancerMetrics struct {
	Range       string                    `json:"range"`
	StepMinutes int                       `json:"step_minutes"`
	Points      []LoadBalancerMetricPoint `json:"points"`
	Totals      struct {
		Requests    int64   `json:"requests"`
		ErrorsPct   float64 `json:"errors_pct"`
		Bytes       int64   `json:"bytes"`
		SessionsMax int64   `json:"sessions_max"`
	} `json:"totals"`
}

type LoadBalancerMetricsResponse struct {
	Data LoadBalancerMetrics `json:"data"`
}

func (c *Client) ListFirewalls(ctx context.Context) (*FirewallListResponse, error) {
	var result FirewallListResponse
	err := c.networkRequest(ctx, http.MethodGet, "/firewalls", nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) GetFirewall(ctx context.Context, id int64) (*FirewallResponse, error) {
	var result FirewallResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("firewalls", id), nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) CreateFirewall(ctx context.Context, name string, rules []FirewallRule) (*FirewallResponse, error) {
	var result FirewallResponse
	body := map[string]any{"name": name}
	if rules != nil {
		body["rules"] = rules
	}
	err := c.networkRequest(ctx, http.MethodPost, "/firewalls", body, &result, http.StatusCreated)
	return &result, err
}

func (c *Client) UpdateFirewall(ctx context.Context, id int64, name string, rules []FirewallRule) (*FirewallResponse, error) {
	var result FirewallResponse
	body := map[string]any{"name": name}
	if rules != nil {
		body["rules"] = rules
	}
	err := c.networkRequest(ctx, http.MethodPut, networkIDPath("firewalls", id), body, &result, http.StatusOK)
	return &result, err
}

func (c *Client) DeleteFirewall(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, networkIDPath("firewalls", id), nil, nil, http.StatusNoContent)
}

func (c *Client) AttachFirewall(ctx context.Context, firewallID, vmID int64) (*FirewallMember, error) {
	var result struct {
		Data FirewallMember `json:"data"`
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("firewalls", firewallID)+"/members", map[string]int64{"vm": vmID}, &result, http.StatusOK, http.StatusCreated)
	return &result.Data, err
}

func (c *Client) DetachFirewall(ctx context.Context, firewallID, vmID int64) error {
	path := networkIDPath("firewalls", firewallID) + "/members/" + strconv.FormatInt(vmID, 10)
	return c.networkRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
}

func (c *Client) ListLoadBalancers(ctx context.Context) (*LoadBalancerListResponse, error) {
	var result LoadBalancerListResponse
	err := c.networkRequest(ctx, http.MethodGet, "/load-balancers", nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) GetLoadBalancer(ctx context.Context, id int64) (*LoadBalancerResponse, error) {
	var result LoadBalancerResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("load-balancers", id), nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) CreateLoadBalancer(ctx context.Context, name string, vpcID int64, idempotencyKey string) (*LoadBalancerResponse, error) {
	var result LoadBalancerResponse
	path := "/load-balancers"
	err := c.networkRequestWithHeaders(ctx, http.MethodPost, path, map[string]any{"name": name, "vpc": vpcID}, &result, []int{http.StatusAccepted}, map[string]string{"Idempotency-Key": idempotencyKey})
	return &result, err
}

func (c *Client) RenameLoadBalancer(ctx context.Context, id int64, name string) (*LoadBalancerResponse, error) {
	var result LoadBalancerResponse
	err := c.networkRequest(ctx, http.MethodPatch, networkIDPath("load-balancers", id), map[string]string{"name": name}, &result, http.StatusOK)
	return &result, err
}

func (c *Client) DeleteLoadBalancer(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, networkIDPath("load-balancers", id), nil, nil, http.StatusAccepted)
}

func (c *Client) GetLoadBalancerMetrics(ctx context.Context, id int64, timeframe string) (*LoadBalancerMetricsResponse, error) {
	path := networkIDPath("load-balancers", id) + "/metrics"
	if timeframe != "" {
		path += "?range=" + url.QueryEscape(timeframe)
	}
	var result LoadBalancerMetricsResponse
	err := c.networkRequest(ctx, http.MethodGet, path, nil, &result, http.StatusOK)
	return &result, err
}

func (c *Client) CreateLoadBalancerPool(ctx context.Context, lbID int64, input map[string]any) (*LoadBalancerPool, error) {
	var result struct {
		Data LoadBalancerPool `json:"data"`
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("load-balancers", lbID)+"/pools", input, &result, http.StatusCreated)
	return &result.Data, err
}

func (c *Client) UpdateLoadBalancerPool(ctx context.Context, lbID, poolID int64, input map[string]any) (*LoadBalancerPool, error) {
	var result struct {
		Data LoadBalancerPool `json:"data"`
	}
	path := networkIDPath("load-balancers", lbID) + "/pools/" + strconv.FormatInt(poolID, 10)
	err := c.networkRequest(ctx, http.MethodPatch, path, input, &result, http.StatusOK)
	return &result.Data, err
}

func (c *Client) DeleteLoadBalancerPool(ctx context.Context, lbID, poolID int64) error {
	path := networkIDPath("load-balancers", lbID) + "/pools/" + strconv.FormatInt(poolID, 10)
	return c.networkRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
}

func (c *Client) ReplaceLoadBalancerTargets(ctx context.Context, lbID, poolID int64, targets []LoadBalancerTarget) (*LoadBalancerPool, error) {
	var result struct {
		Data LoadBalancerPool `json:"data"`
	}
	path := networkIDPath("load-balancers", lbID) + "/pools/" + strconv.FormatInt(poolID, 10) + "/targets"
	err := c.networkRequest(ctx, http.MethodPut, path, map[string]any{"targets": targets}, &result, http.StatusOK)
	return &result.Data, err
}

func (c *Client) UpdateLoadBalancerTarget(ctx context.Context, lbID, poolID, targetID int64, input map[string]any) (*LoadBalancerTarget, error) {
	var result struct {
		Data LoadBalancerTarget `json:"data"`
	}
	path := networkIDPath("load-balancers", lbID) + "/pools/" + strconv.FormatInt(poolID, 10) + "/targets/" + strconv.FormatInt(targetID, 10)
	err := c.networkRequest(ctx, http.MethodPatch, path, input, &result, http.StatusOK)
	return &result.Data, err
}

func (c *Client) DeleteLoadBalancerTarget(ctx context.Context, lbID, poolID, targetID int64) error {
	path := networkIDPath("load-balancers", lbID) + "/pools/" + strconv.FormatInt(poolID, 10) + "/targets/" + strconv.FormatInt(targetID, 10)
	return c.networkRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
}

func (c *Client) CreateLoadBalancerListener(ctx context.Context, lbID int64, input map[string]any) (*LoadBalancerListener, error) {
	var result struct {
		Data LoadBalancerListener `json:"data"`
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("load-balancers", lbID)+"/listeners", input, &result, http.StatusCreated)
	return &result.Data, err
}

func (c *Client) UpdateLoadBalancerListener(ctx context.Context, lbID, listenerID int64, input map[string]any) (*LoadBalancerListener, error) {
	var result struct {
		Data LoadBalancerListener `json:"data"`
	}
	path := networkIDPath("load-balancers", lbID) + "/listeners/" + strconv.FormatInt(listenerID, 10)
	err := c.networkRequest(ctx, http.MethodPatch, path, input, &result, http.StatusOK)
	return &result.Data, err
}

func (c *Client) DeleteLoadBalancerListener(ctx context.Context, lbID, listenerID int64) error {
	path := networkIDPath("load-balancers", lbID) + "/listeners/" + strconv.FormatInt(listenerID, 10)
	return c.networkRequest(ctx, http.MethodDelete, path, nil, nil, http.StatusNoContent)
}

func (c *Client) ReplaceLoadBalancerRules(ctx context.Context, lbID, listenerID int64, rules []LoadBalancerRule) (*LoadBalancerListener, error) {
	var result struct {
		Data LoadBalancerListener `json:"data"`
	}
	path := networkIDPath("load-balancers", lbID) + "/listeners/" + strconv.FormatInt(listenerID, 10) + "/rules"
	err := c.networkRequest(ctx, http.MethodPut, path, map[string]any{"rules": rules}, &result, http.StatusOK)
	return &result.Data, err
}

func networkIDPath(resource string, id int64) string {
	return "/" + resource + "/" + strconv.FormatInt(id, 10)
}

func (c *Client) networkRequest(ctx context.Context, method, path string, body, target any, expected ...int) error {
	return c.networkRequestWithHeaders(ctx, method, path, body, target, expected, nil)
}

func (c *Client) networkRequestWithHeaders(ctx context.Context, method, path string, body, target any, expected []int, headers map[string]string) error {
	var requestBody *bytes.Reader
	if body == nil {
		requestBody = bytes.NewReader(nil)
	} else {
		encoded, err := json.Marshal(body)
		if err != nil {
			return err
		}
		requestBody = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, requestBody)
	if err != nil {
		return err
	}
	request.Header.Set("Authorization", "Bearer "+c.Token)
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	for _, status := range expected {
		if response.StatusCode == status {
			if target == nil || status == http.StatusNoContent {
				return nil
			}
			if err := json.NewDecoder(response.Body).Decode(target); err != nil {
				return err
			}
			return nil
		}
	}
	if response.StatusCode == http.StatusNotFound {
		return ErrNetworkResourceNotFound
	}
	return fmt.Errorf("API HTTP error %d during %s %s", response.StatusCode, method, path)
}
