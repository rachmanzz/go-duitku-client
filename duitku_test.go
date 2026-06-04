package duitku

import (
	"encoding/json"
	"net/url"
	"testing"
)

func TestGenerateSignature(t *testing.T) {
	sig := generateSignature("DXXXX", "1773728479616", "testkey")
	if sig == "" {
		t.Error("signature should not be empty")
	}
}

func TestVerifyCallbackSignature(t *testing.T) {
	apiKey := "testapikey"
	sig := hmacSHA256("DXXXX150000abcde12345", apiKey)
	if !VerifyCallbackSignature("DXXXX", 150000, "abcde12345", apiKey, sig) {
		t.Error("signature verification should pass")
	}
}

func TestVerifyCallbackSignature_FailsOnWrongKey(t *testing.T) {
	if VerifyCallbackSignature("DXXXX", 150000, "abcde12345", "wrongkey", "somesig") {
		t.Error("should fail with wrong key")
	}
}

func TestCreateInvoiceRequest_Marshal(t *testing.T) {
	req := &CreateInvoiceRequest{
		PaymentAmount:   40000,
		MerchantOrderID: "1648542419",
		ProductDetails:  "Test Pay with duitku",
		Email:           "test@example.com",
		PhoneNumber:     "08123456789",
		CustomerVaName:  "John Doe",
		CallbackURL:     "https://example.com/callback",
		ReturnURL:       "https://example.com/return",
		ExpiryPeriod:    10,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	if len(data) == 0 {
		t.Error("marshaled data should not be empty")
	}
}

func TestParseCallback(t *testing.T) {
	form := url.Values{}
	form.Set("merchantCode", "DXXXX")
	form.Set("amount", "150000")
	form.Set("merchantOrderId", "abcde12345")
	form.Set("resultCode", "00")
	form.Set("reference", "REF123")
	form.Set("signature", "testsig")

	req, err := ParseCallback(form)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	if req.MerchantCode != "DXXXX" {
		t.Errorf("expected DXXXX, got %s", req.MerchantCode)
	}
	if req.Amount != 150000 {
		t.Errorf("expected 150000, got %d", req.Amount)
	}
	if req.ResultCode != "00" {
		t.Errorf("expected 00, got %s", req.ResultCode)
	}
}

func TestParseCallbackResult(t *testing.T) {
	if ParseCallbackResult("00") != CallbackResultSuccess {
		t.Error("00 should be success")
	}
	if ParseCallbackResult("01") != CallbackResultFailed {
		t.Error("01 should be failed")
	}
	if ParseCallbackResult("99") != CallbackResultUnknown {
		t.Error("99 should be unknown")
	}
}

func TestNewClient(t *testing.T) {
	c := NewClient("DXXXX", "testkey")
	if c.MerchantCode != "DXXXX" {
		t.Errorf("expected DXXXX, got %s", c.MerchantCode)
	}
	if c.APIKey != "testkey" {
		t.Errorf("expected testkey, got %s", c.APIKey)
	}
	if c.BaseURL() != BaseURLSandbox {
		t.Errorf("expected sandbox default")
	}
}

func TestNewClient_WithBaseURL(t *testing.T) {
	c := NewClient("DXXXX", "testkey", WithBaseURL(BaseURLProduction))
	if c.BaseURL() != BaseURLProduction {
		t.Errorf("expected production URL")
	}
}
