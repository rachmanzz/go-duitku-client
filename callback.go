package duitku

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
)

var ErrInvalidSignature = errors.New("duitku: invalid callback signature")

type CallbackHandler func(req *CallbackRequest) error

type CallbackResult int

const (
	CallbackResultSuccess CallbackResult = iota
	CallbackResultFailed
	CallbackResultUnknown
)

func ParseCallbackResult(code string) CallbackResult {
	switch code {
	case "00":
		return CallbackResultSuccess
	case "01":
		return CallbackResultFailed
	default:
		return CallbackResultUnknown
	}
}

func ParseCallbackFromMap(data map[string]any) *CallbackRequest {
	getStr := func(k string) string {
		v, ok := data[k]
		if !ok {
			return ""
		}
		s, _ := v.(string)
		return s
	}
	getInt64 := func(k string) int64 {
		switch v := data[k].(type) {
		case float64:
			return int64(v)
		case string:
			n, _ := strconv.ParseInt(v, 10, 64)
			return n
		}
		return 0
	}

	return &CallbackRequest{
		MerchantCode:           getStr("merchantCode"),
		Amount:                 getInt64("amount"),
		MerchantOrderID:        getStr("merchantOrderId"),
		ProductDetail:          getStr("productDetail"),
		AdditionalParam:        getStr("additionalParam"),
		PaymentCode:            getStr("paymentCode"),
		ResultCode:             getStr("resultCode"),
		MerchantUserID:         getStr("merchantUserId"),
		Reference:              getStr("reference"),
		Signature:              getStr("signature"),
		PublisherOrderID:       getStr("publisherOrderId"),
		SpUserHash:             getStr("spUserHash"),
		SettlementDate:         getStr("settlementDate"),
		IssuerCode:             getStr("issuerCode"),
		BankAppCode:            getStr("bankAppCode"),
		BankOrderID:            getStr("bankOrderId"),
		BankRespCode:           getStr("bankRespCode"),
		BankRespMsg:            getStr("bankRespMsg"),
		CardName:               getStr("cardName"),
		CardType:               getStr("cardType"),
		MaskedNumber:           getStr("maskedNumber"),
		TokenID:                getStr("tokenId"),
		TransactionState:       getStr("transactionState"),
		TransactionStateStatus: getStr("transactionStateStatus"),
		MerchantCustomerID:     getStr("merchantCustomerId"),
		ExpiryDate:             getStr("expiryDate"),
		CustomerName:           getStr("customerName"),
	}
}

func ParseCallback(form url.Values) (*CallbackRequest, error) {
	amountStr := form.Get("amount")
	var amount int64
	if amountStr != "" {
		var err error
		amount, err = strconv.ParseInt(amountStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("duitku: parse amount: %w", err)
		}
	}

	return &CallbackRequest{
		MerchantCode:           form.Get("merchantCode"),
		Amount:                 amount,
		MerchantOrderID:        form.Get("merchantOrderId"),
		ProductDetail:          form.Get("productDetail"),
		AdditionalParam:        form.Get("additionalParam"),
		PaymentCode:            form.Get("paymentCode"),
		ResultCode:             form.Get("resultCode"),
		MerchantUserID:         form.Get("merchantUserId"),
		Reference:              form.Get("reference"),
		Signature:              form.Get("signature"),
		PublisherOrderID:       form.Get("publisherOrderId"),
		SpUserHash:             form.Get("spUserHash"),
		SettlementDate:         form.Get("settlementDate"),
		IssuerCode:             form.Get("issuerCode"),
		BankAppCode:            form.Get("bankAppCode"),
		BankOrderID:            form.Get("bankOrderId"),
		BankRespCode:           form.Get("bankRespCode"),
		BankRespMsg:            form.Get("bankRespMsg"),
		CardName:               form.Get("cardName"),
		CardType:               form.Get("cardType"),
		MaskedNumber:           form.Get("maskedNumber"),
		TokenID:                form.Get("tokenId"),
		TransactionState:       form.Get("transactionState"),
		TransactionStateStatus: form.Get("transactionStateStatus"),
		MerchantCustomerID:     form.Get("merchantCustomerId"),
		ExpiryDate:             form.Get("expiryDate"),
		CustomerName:           form.Get("customerName"),
	}, nil
}

func (c *Client) VerifyCallback(req *CallbackRequest) bool {
	if req.MerchantCode == "" || req.Amount == 0 || req.MerchantOrderID == "" || req.Signature == "" {
		return false
	}
	return VerifyCallbackSignature(req.MerchantCode, req.Amount, req.MerchantOrderID, c.APIKey, req.Signature)
}

func (c *Client) ProcessCallback(form url.Values, handler CallbackHandler) error {
	cb, err := ParseCallback(form)
	if err != nil {
		return err
	}
	if !c.VerifyCallback(cb) {
		return ErrInvalidSignature
	}
	return handler(cb)
}
