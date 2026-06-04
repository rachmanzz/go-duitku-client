package duitku

import (
	"net/http"
	"time"
)

const (
	BaseURLSandbox    = "https://api-sandbox.duitku.com"
	BaseURLProduction = "https://api-prod.duitku.com"
)

type Client struct {
	MerchantCode string
	APIKey       string
	baseURL      string
	httpClient   *http.Client
}

type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

func NewClient(merchantCode, apiKey string, opts ...Option) *Client {
	c := &Client{
		MerchantCode: merchantCode,
		APIKey:       apiKey,
		baseURL:      BaseURLSandbox,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

func (c *Client) BaseURL() string {
	return c.baseURL
}
