package duitku

import (
	"strconv"
	"strings"
)

// Int64 handles JSON unmarshaling from both number and string formats.
type Int64 int64

func (n *Int64) UnmarshalJSON(data []byte) error {
	s := strings.Trim(string(data), `"`)
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return err
	}
	*n = Int64(v)
	return nil
}

type Address struct {
	FirstName   string `json:"firstName,omitempty"`
	LastName    string `json:"lastName,omitempty"`
	Address     string `json:"address,omitempty"`
	City        string `json:"city,omitempty"`
	PostalCode  string `json:"postalCode,omitempty"`
	Phone       string `json:"phone,omitempty"`
	CountryCode string `json:"countryCode,omitempty"`
}

type CustomerDetail struct {
	FirstName          string  `json:"firstName,omitempty"`
	LastName           string  `json:"lastName,omitempty"`
	Email              string  `json:"email,omitempty"`
	PhoneNumber        string  `json:"phoneNumber,omitempty"`
	BillingAddress     *Address `json:"billingAddress,omitempty"`
	ShippingAddress    *Address `json:"shippingAddress,omitempty"`
	MerchantCustomerID string  `json:"merchantCustomerId,omitempty"`
}

type ItemDetail struct {
	Name     string `json:"name"`
	Price    int64  `json:"price"`
	Quantity int    `json:"quantity"`
}

type CreditCardDetail struct {
	SaveCardToken int      `json:"saveCardToken,omitempty"`
	Acquirer     string   `json:"acquirer,omitempty"`
	BinWhitelist []string `json:"binWhitelist,omitempty"`
}

type CreateInvoiceRequest struct {
	PaymentAmount    int64             `json:"paymentAmount"`
	MerchantOrderID  string            `json:"merchantOrderId"`
	ProductDetails   string            `json:"productDetails"`
	Email            string            `json:"email"`
	PhoneNumber      string            `json:"phoneNumber,omitempty"`
	AdditionalParam  string            `json:"additionalParam,omitempty"`
	MerchantUserInfo string            `json:"merchantUserInfo,omitempty"`
	CustomerVaName   string            `json:"customerVaName,omitempty"`
	PaymentMethod    string            `json:"paymentMethod,omitempty"`
	ItemDetails      []ItemDetail      `json:"itemDetails,omitempty"`
	CustomerDetail   *CustomerDetail   `json:"customerDetail,omitempty"`
	CreditCardDetail *CreditCardDetail `json:"creditCardDetail,omitempty"`
	CallbackURL      string            `json:"callbackUrl"`
	ReturnURL        string            `json:"returnUrl"`
	ExpiryPeriod     int               `json:"expiryPeriod,omitempty"`
}

type CreateInvoiceResponse struct {
	MerchantCode  string `json:"merchantCode"`
	Reference     string `json:"reference"`
	PaymentURL    string `json:"paymentUrl"`
	StatusCode    string `json:"statusCode"`
	StatusMessage string `json:"statusMessage"`
	VaNumber      string `json:"vaNumber,omitempty"`
	Amount        Int64  `json:"amount,omitempty"`
}

type CallbackRequest struct {
	MerchantCode         string `form:"merchantCode"`
	Amount               int64  `form:"amount"`
	MerchantOrderID      string `form:"merchantOrderId"`
	ProductDetail        string `form:"productDetail"`
	AdditionalParam      string `form:"additionalParam"`
	PaymentCode          string `form:"paymentCode"`
	ResultCode           string `form:"resultCode"`
	MerchantUserID       string `form:"merchantUserId"`
	Reference            string `form:"reference"`
	Signature            string `form:"signature"`
	PublisherOrderID     string `form:"publisherOrderId"`
	SpUserHash           string `form:"spUserHash"`
	SettlementDate       string `form:"settlementDate"`
	IssuerCode           string `form:"issuerCode"`
	BankAppCode          string `form:"bankAppCode"`
	BankOrderID          string `form:"bankOrderId"`
	BankRespCode         string `form:"bankRespCode"`
	BankRespMsg          string `form:"bankRespMsg"`
	CardName             string `form:"cardName"`
	CardType             string `form:"cardType"`
	MaskedNumber         string `form:"maskedNumber"`
	TokenID              string `form:"tokenId"`
	TransactionState     string `form:"transactionState"`
	TransactionStateStatus string `form:"transactionStateStatus"`
	MerchantCustomerID   string `form:"merchantCustomerId"`
	ExpiryDate           string `form:"expiryDate"`
}
