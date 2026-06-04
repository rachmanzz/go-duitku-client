package duitku

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type InvoiceError struct {
	HTTPCode int
	Body     string
}

func (e *InvoiceError) Error() string {
	return fmt.Sprintf("duitku: create invoice failed with HTTP %d: %s", e.HTTPCode, e.Body)
}

func (c *Client) CreateInvoice(ctx context.Context, req *CreateInvoiceRequest) (*CreateInvoiceResponse, error) {
	timestamp := fmt.Sprintf("%d", time.Now().UnixMilli())
	signature := generateSignature(c.MerchantCode, timestamp, c.APIKey)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("duitku: marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL()+"/api/merchant/createInvoice", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("duitku: create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-duitku-signature", signature)
	httpReq.Header.Set("x-duitku-timestamp", timestamp)
	httpReq.Header.Set("x-duitku-merchantcode", c.MerchantCode)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("duitku: execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("duitku: read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, &InvoiceError{
			HTTPCode: resp.StatusCode,
			Body:     string(respBody),
		}
	}

	var result CreateInvoiceResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("duitku: unmarshal response: %w", err)
	}

	return &result, nil
}
