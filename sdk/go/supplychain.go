package sdk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	BaseURL    string
	APIKey     string
	ClientType string
	HTTP       *http.Client
}

func New(baseURL, apiKey, clientType string) *Client {
	if baseURL == "" {
		baseURL = "https://api.supplychain.example.com"
	}
	return &Client{BaseURL: baseURL, APIKey: apiKey, ClientType: clientType, HTTP: http.DefaultClient}
}

func (c *Client) headers(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	if c.ClientType != "" {
		req.Header.Set("X-Client-Type", c.ClientType)
	}
}

func (c *Client) Alerts(params map[string]string) (*http.Response, error) {
	u, _ := url.Parse(c.BaseURL + "/v1/alerts")
	q := u.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	u.RawQuery = q.Encode()
	req, _ := http.NewRequest("GET", u.String(), nil)
	c.headers(req)
	return c.HTTP.Do(req)
}

func (c *Client) Usage() (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", c.BaseURL+"/v1/usage", nil)
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) UsageTimeseries(bucket, start, end string) (map[string]interface{}, error) {
	u := fmt.Sprintf("%s/v1/usage/timeseries?bucket=%s&start=%s&end=%s", c.BaseURL, url.QueryEscape(bucket), url.QueryEscape(start), url.QueryEscape(end))
	req, _ := http.NewRequest("GET", u, nil)
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var out map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateCheckoutSession(planCode, interval string, overage bool) (string, error) {
	body := fmt.Sprintf(`{"plan_code":"%s","interval":"%s","overage_enabled":%t}`, planCode, interval, overage)
	req, _ := http.NewRequest("POST", c.BaseURL+"/v1/billing/checkout-session", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.URL, nil
}

func (c *Client) CreatePortalSession() (string, error) {
	req, _ := http.NewRequest("POST", c.BaseURL+"/v1/billing/portal-session", nil)
	c.headers(req)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.URL, nil
}
