package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

type AccountAreaMoney string

func (m *AccountAreaMoney) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		*m = AccountAreaMoney(text)
		return nil
	}
	var number json.Number
	if err := json.Unmarshal(data, &number); err != nil {
		return fmt.Errorf("decode monetary amount: %w", err)
	}
	*m = AccountAreaMoney(number.String())
	return nil
}

type AccountAreaPlansResponse struct {
	Data []Plan `json:"data"`
}
type AccountAreaImagesResponse struct {
	Data []Image `json:"data"`
}
type AccountAreaVPCListResponse struct {
	Data  []AccountAreaVPC `json:"data"`
	Quota int              `json:"quota"`
}
type AccountAreaVPCResponse struct {
	Data AccountAreaVPC `json:"data"`
}
type AccountAreaVPC struct {
	ID              int                    `json:"id"`
	Name            string                 `json:"name"`
	CIDR            string                 `json:"cidr"`
	Region          string                 `json:"region"`
	Status          string                 `json:"status"`
	Gateway         string                 `json:"gateway"`
	InternetGateway bool                   `json:"internet_gateway"`
	Members         []AccountAreaVPCMember `json:"members"`
	CreatedAt       string                 `json:"created_at"`
}
type AccountAreaVPCMember struct {
	VM            int    `json:"vm"`
	Hostname      string `json:"hostname"`
	IP            string `json:"ip"`
	Configuration string `json:"configuration"`
}
type AccountAreaSSHKey struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
	PublicKey   string `json:"public_key"`
	CreatedAt   string `json:"created_at"`
}
type AccountAreaSSHKeysResponse struct {
	Data []AccountAreaSSHKey `json:"data"`
}
type AccountAreaSSHKeyResponse struct {
	Data AccountAreaSSHKey `json:"data"`
}
type AccountAreaMoneyBuckets struct {
	Balance         AccountAreaMoney `json:"balance"`
	Paid            AccountAreaMoney `json:"paid"`
	Gifted          AccountAreaMoney `json:"gifted"`
	GiftedExpiresAt *string          `json:"gifted_expires_at"`
}
type AccountAreaHourly struct {
	BurnRate       AccountAreaMoney `json:"burn_rate"`
	RunwayHours    *float64         `json:"runway_hours"`
	ActiveMachines int              `json:"active_machines"`
}
type AccountAreaBudget struct {
	Monthly     *AccountAreaMoney `json:"monthly"`
	MonthToDate AccountAreaMoney  `json:"month_to_date"`
	Percent     *float64          `json:"percent"`
	CapPercent  int               `json:"cap_percent"`
}
type AccountAreaQuota struct {
	Limit     int `json:"limit"`
	Used      int `json:"used"`
	Remaining int `json:"remaining"`
}
type AccountAreaAutoTopup struct {
	Enabled   bool              `json:"enabled"`
	Threshold *AccountAreaMoney `json:"threshold"`
	Amount    *AccountAreaMoney `json:"amount"`
}
type AccountAreaAccount struct {
	Currency    string                  `json:"currency"`
	BillingMode string                  `json:"billing_mode"`
	Credit      AccountAreaMoneyBuckets `json:"credit"`
	Hourly      AccountAreaHourly       `json:"hourly"`
	Budget      AccountAreaBudget       `json:"budget"`
	Quota       AccountAreaQuota        `json:"quota"`
	AutoTopup   AccountAreaAutoTopup    `json:"auto_topup"`
}
type AccountAreaAccountResponse struct {
	Data AccountAreaAccount `json:"data"`
}
type AccountAreaPageMeta struct {
	NextCursor *string `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
}
type AccountAreaUsageRecord struct {
	VMID        *int             `json:"vm_id"`
	Hostname    *string          `json:"hostname"`
	Kind        string           `json:"kind"`
	PeriodStart string           `json:"period_start"`
	Hours       int              `json:"hours"`
	Rate        AccountAreaMoney `json:"rate"`
	RateUnit    string           `json:"rate_unit"`
	Amount      AccountAreaMoney `json:"amount"`
	Currency    string           `json:"currency"`
}
type AccountAreaUsagePage struct {
	Data []AccountAreaUsageRecord `json:"data"`
	Meta struct {
		AccountAreaPageMeta
		Total    AccountAreaMoney `json:"total"`
		Currency string           `json:"currency"`
	} `json:"meta"`
}
type AccountAreaUsageResponse struct {
	Data     []AccountAreaUsageRecord
	Total    AccountAreaMoney
	Currency string
}
type AccountAreaUsageQuery struct {
	From, To, GroupBy string
	VMID              *int64
}
type AccountAreaRunnerSize struct {
	Label          string           `json:"label"`
	VCPU           int              `json:"vcpu"`
	MemoryGB       int              `json:"memory_gb"`
	Image          string           `json:"image"`
	PricePerMinute AccountAreaMoney `json:"price_per_minute"`
	Currency       string           `json:"currency"`
}
type AccountAreaRunnerInstallation struct {
	ID                  int     `json:"id"`
	Account             string  `json:"account"`
	Type                string  `json:"type"`
	RepositorySelection *string `json:"repository_selection"`
	Status              string  `json:"status"`
	LinkedAt            *string `json:"linked_at"`
}
type AccountAreaRunnerRepository struct {
	Repository *string          `json:"repository"`
	VCPU       int              `json:"vcpu"`
	Jobs       int              `json:"jobs"`
	Minutes    int              `json:"minutes"`
	Amount     AccountAreaMoney `json:"amount"`
}
type AccountAreaRunnerMonth struct {
	PeriodStart          string                        `json:"period_start"`
	Minutes              int                           `json:"minutes"`
	FreeMinutesUsed      int                           `json:"free_minutes_used"`
	FreeMinutesAllowance int                           `json:"free_minutes_allowance"`
	Amount               AccountAreaMoney              `json:"amount"`
	CacheAmount          AccountAreaMoney              `json:"cache_amount"`
	Currency             string                        `json:"currency"`
	Repositories         []AccountAreaRunnerRepository `json:"repositories"`
}
type AccountAreaCacheRepository struct {
	Repository   *string `json:"repository"`
	RepositoryID int     `json:"repository_id"`
	Bytes        int64   `json:"bytes"`
	Entries      int     `json:"entries"`
	LastUsedAt   *string `json:"last_used_at"`
	Emptying     bool    `json:"emptying"`
}
type AccountAreaRunnerCache struct {
	PricePerGBMonth      AccountAreaMoney             `json:"price_per_gb_month"`
	Currency             string                       `json:"currency"`
	Bytes                int64                        `json:"bytes"`
	RepositoryLimitGBMax int                          `json:"repository_limit_gb_max"`
	RetentionDays        int                          `json:"retention_days"`
	Repositories         []AccountAreaCacheRepository `json:"repositories"`
}
type AccountAreaDockerCacheRepository struct {
	Repository    *string `json:"repository"`
	RepositoryID  int     `json:"repository_id"`
	Enabled       bool    `json:"enabled"`
	Bytes         int64   `json:"bytes"`
	LastSuccessAt *string `json:"last_success_at"`
	LastUsedAt    *string `json:"last_used_at"`
	Emptying      bool    `json:"emptying"`
}
type AccountAreaDockerCache struct {
	RepositorySizeGB int                                `json:"repository_size_gb"`
	RetentionDays    int                                `json:"retention_days"`
	Repositories     []AccountAreaDockerCacheRepository `json:"repositories"`
}
type AccountAreaRunners struct {
	Sizes       []AccountAreaRunnerSize `json:"sizes"`
	Concurrency struct {
		MaxVCPU     int `json:"max_vcpu"`
		RunningVCPU int `json:"running_vcpu"`
	} `json:"concurrency"`
	Installations []AccountAreaRunnerInstallation `json:"installations"`
	Month         AccountAreaRunnerMonth          `json:"month"`
	Cache         AccountAreaRunnerCache          `json:"cache"`
	DockerCache   AccountAreaDockerCache          `json:"docker_cache"`
}
type AccountAreaRunnersResponse struct {
	Data AccountAreaRunners `json:"data"`
}
type AccountAreaRunnerJob struct {
	ID               int              `json:"id"`
	GitHubJobID      int64            `json:"github_job_id"`
	RunID            int64            `json:"run_id"`
	RunAttempt       int              `json:"run_attempt"`
	Repository       string           `json:"repository"`
	Workflow         *string          `json:"workflow"`
	Job              *string          `json:"job"`
	Label            string           `json:"label"`
	VCPU             int              `json:"vcpu"`
	Status           string           `json:"status"`
	WaitingReason    *string          `json:"waiting_reason"`
	Refusal          *string          `json:"refusal"`
	Conclusion       *string          `json:"conclusion"`
	StoppedForCredit bool             `json:"stopped_for_credit"`
	QueuedAt         *string          `json:"queued_at"`
	StartedAt        *string          `json:"started_at"`
	CompletedAt      *string          `json:"completed_at"`
	Minutes          *int             `json:"minutes"`
	FreeMinutes      int              `json:"free_minutes"`
	Amount           AccountAreaMoney `json:"amount"`
	Currency         *string          `json:"currency"`
	URL              string           `json:"url"`
}
type AccountAreaRunnerJobsPage struct {
	Data []AccountAreaRunnerJob `json:"data"`
	Meta AccountAreaPageMeta    `json:"meta"`
}
type AccountAreaRunnerJobsResponse struct{ Data []AccountAreaRunnerJob }

func (c *Client) AccountAreaListPlans(ctx context.Context, region string) (*AccountAreaPlansResponse, error) {
	var result AccountAreaPlansResponse
	err := c.networkRequest(ctx, http.MethodGet, accountAreaQueryPath("/plans", map[string]string{"region": region}), nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaListImages(ctx context.Context, filters map[string]string) (*AccountAreaImagesResponse, error) {
	var result AccountAreaImagesResponse
	err := c.networkRequest(ctx, http.MethodGet, accountAreaQueryPath("/images", filters), nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaListVPCs(ctx context.Context) (*AccountAreaVPCListResponse, error) {
	var result AccountAreaVPCListResponse
	err := c.networkRequest(ctx, http.MethodGet, "/vpcs", nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaGetVPC(ctx context.Context, id int64) (*AccountAreaVPCResponse, error) {
	var result AccountAreaVPCResponse
	err := c.networkRequest(ctx, http.MethodGet, networkIDPath("vpcs", id), nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaCreateVPCMember(ctx context.Context, vpcID, vmID int64) (*AccountAreaVPCMember, error) {
	var result struct {
		Data AccountAreaVPCMember `json:"data"`
	}
	err := c.networkRequest(ctx, http.MethodPost, networkIDPath("vpcs", vpcID)+"/members", map[string]int64{"vm": vmID}, &result, http.StatusOK, http.StatusCreated)
	return &result.Data, err
}
func (c *Client) AccountAreaMoveVPCMember(ctx context.Context, vpcID, vmID, toID int64) (*AccountAreaVPCMember, error) {
	var result struct {
		Data AccountAreaVPCMember `json:"data"`
	}
	path := networkIDPath("vpcs", vpcID) + "/members/" + strconv.FormatInt(vmID, 10) + "/move"
	err := c.networkRequest(ctx, http.MethodPost, path, map[string]int64{"to": toID}, &result, http.StatusOK)
	return &result.Data, err
}
func (c *Client) AccountAreaDeleteVPCMember(ctx context.Context, vpcID, vmID int64) error {
	return c.networkRequest(ctx, http.MethodDelete, networkIDPath("vpcs", vpcID)+"/members/"+strconv.FormatInt(vmID, 10), nil, nil, http.StatusNoContent)
}
func (c *Client) AccountAreaListSSHKeys(ctx context.Context) (*AccountAreaSSHKeysResponse, error) {
	var result AccountAreaSSHKeysResponse
	err := c.networkRequest(ctx, http.MethodGet, "/ssh-keys", nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaCreateSSHKey(ctx context.Context, name, publicKey string) (*AccountAreaSSHKeyResponse, error) {
	var result AccountAreaSSHKeyResponse
	err := c.networkRequest(ctx, http.MethodPost, "/ssh-keys", map[string]string{"name": name, "public_key": publicKey}, &result, http.StatusCreated)
	return &result, err
}
func (c *Client) AccountAreaDeleteSSHKey(ctx context.Context, id int64) error {
	return c.networkRequest(ctx, http.MethodDelete, "/ssh-keys/"+strconv.FormatInt(id, 10), nil, nil, http.StatusNoContent)
}
func (c *Client) AccountAreaGetAccount(ctx context.Context) (*AccountAreaAccountResponse, error) {
	var result AccountAreaAccountResponse
	err := c.networkRequest(ctx, http.MethodGet, "/account", nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaSetBudget(ctx context.Context, monthly *string) (*AccountAreaBudget, error) {
	var result struct {
		Data AccountAreaBudget `json:"data"`
	}
	var value any
	if monthly != nil {
		value = *monthly
	}
	err := c.networkRequest(ctx, http.MethodPut, "/account/budget", map[string]any{"monthly": value}, &result, http.StatusOK)
	return &result.Data, err
}
func (c *Client) AccountAreaListUsage(ctx context.Context, query AccountAreaUsageQuery) (*AccountAreaUsageResponse, error) {
	params := map[string]string{"from": query.From, "to": query.To, "group_by": query.GroupBy}
	if query.VMID != nil {
		params["vm_id"] = strconv.FormatInt(*query.VMID, 10)
	}
	all := &AccountAreaUsageResponse{}
	cursor := ""
	for {
		params["cursor"], params["limit"] = cursor, "200"
		var page AccountAreaUsagePage
		err := c.networkRequest(ctx, http.MethodGet, accountAreaQueryPath("/usage", params), nil, &page, http.StatusOK)
		if err != nil {
			return nil, err
		}
		all.Data = append(all.Data, page.Data...)
		all.Total, all.Currency = page.Meta.Total, page.Meta.Currency
		if !page.Meta.HasMore {
			return all, nil
		}
		if page.Meta.NextCursor == nil || *page.Meta.NextCursor == "" || *page.Meta.NextCursor == cursor {
			return nil, fmt.Errorf("usage pagination returned has_more without a new cursor")
		}
		cursor = *page.Meta.NextCursor
	}
}
func (c *Client) AccountAreaGetRunners(ctx context.Context) (*AccountAreaRunnersResponse, error) {
	var result AccountAreaRunnersResponse
	err := c.networkRequest(ctx, http.MethodGet, "/runners", nil, &result, http.StatusOK)
	return &result, err
}
func (c *Client) AccountAreaListRunnerJobs(ctx context.Context, repository, status string, limit int) (*AccountAreaRunnerJobsResponse, error) {
	params := map[string]string{"repository": repository, "status": status}
	if limit > 0 {
		params["limit"] = strconv.Itoa(limit)
	}
	all := &AccountAreaRunnerJobsResponse{}
	cursor := ""
	for {
		params["cursor"] = cursor
		var page AccountAreaRunnerJobsPage
		err := c.networkRequest(ctx, http.MethodGet, accountAreaQueryPath("/runners/jobs", params), nil, &page, http.StatusOK)
		if err != nil {
			return nil, err
		}
		all.Data = append(all.Data, page.Data...)
		if !page.Meta.HasMore {
			return all, nil
		}
		if page.Meta.NextCursor == nil || *page.Meta.NextCursor == "" || *page.Meta.NextCursor == cursor {
			return nil, fmt.Errorf("runner job pagination returned has_more without a new cursor")
		}
		cursor = *page.Meta.NextCursor
	}
}
func accountAreaQueryPath(path string, filters map[string]string) string {
	query := url.Values{}
	for key, value := range filters {
		if value != "" {
			query.Set(key, value)
		}
	}
	if encoded := query.Encode(); encoded != "" {
		return path + "?" + encoded
	}
	return path
}
