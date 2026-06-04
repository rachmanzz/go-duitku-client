# duitku-client

Go SDK for [Duitku POP](https://docs.duitku.com/pop/en/) payment gateway — server-side integration.

## Install

```bash
go get github.com/rachmanzz/go-duitku-client
```

## Usage

### Init Client

```go
import "github.com/rachmanzz/duitku-client"

// sandbox (default)
c := duitku.NewClient("DXXXX", "API_KEY")

// production
c := duitku.NewClient("DXXXX", "API_KEY",
    duitku.WithBaseURL(duitku.BaseURLProduction),
)

// custom endpoint
c := duitku.NewClient("DXXXX", "API_KEY",
    duitku.WithBaseURL("https://api-sandbox.duitku.com"),
)

// custom HTTP client
c := duitku.NewClient("DXXXX", "API_KEY",
    duitku.WithHTTPClient(&http.Client{Timeout: 10 * time.Second}),
)
```

### Create Invoice

```go
resp, err := c.CreateInvoice(ctx, &duitku.CreateInvoiceRequest{
    PaymentAmount:   50000,
    MerchantOrderID: "ORDER-001",
    ProductDetails:  "Product A",
    Email:           "customer@mail.com",
    CustomerVaName:  "John Doe",
    ItemDetails: []duitku.ItemDetail{
        {Name: "Item 1", Price: 25000, Quantity: 2},
    },
    CustomerDetail: &duitku.CustomerDetail{
        FirstName: "John",
        LastName:  "Doe",
        Email:     "customer@mail.com",
    },
    CallbackURL: "https://mysite.com/callback",
    ReturnURL:   "https://mysite.com/return",
    ExpiryPeriod: 60,
})

if err != nil {
    // handle error
}
// resp.PaymentURL, resp.Reference, resp.StatusCode ...
```

### Handle Callback

```go
func callbackHandler(w http.ResponseWriter, r *http.Request) {
    r.ParseForm()
    err := c.ProcessCallback(r.PostForm, func(cb *duitku.CallbackRequest) error {
        switch duitku.ParseCallbackResult(cb.ResultCode) {
        case duitku.CallbackResultSuccess:
            // update order status = paid
        case duitku.CallbackResultFailed:
            // update order status = failed
        }
        return nil
    })
    if err != nil {
        http.Error(w, err.Error(), 400)
        return
    }
    w.WriteHeader(200)
}
```

### Framework Adapters

#### Gin

```go
import "github.com/rachmanzz/duitku-client/callbackgin"

r.POST("/callback", func(c *gin.Context) {
    callbackgin.Handle(c, client, func(cb *duitku.CallbackRequest) error {
        // update order status based on cb.ResultCode
        return nil
    })
})
```

#### Echo

```go
import "github.com/rachmanzz/duitku-client/callbackecho"

e.POST("/callback", func(c echo.Context) error {
    return callbackecho.Handle(c, client, func(cb *duitku.CallbackRequest) error {
        // update order status based on cb.ResultCode
        return nil
    })
})
```

#### Fiber

```go
import "github.com/rachmanzz/duitku-client/callbackfiber"

app.Post("/callback", func(c *fiber.Ctx) error {
    return callbackfiber.Handle(c, client, func(cb *duitku.CallbackRequest) error {
        // update order status based on cb.ResultCode
        return nil
    })
})
```

#### Goravel

```go
import "github.com/rachmanzz/duitku-client/callbackgoravel"

func (c *Controller) Callback(ctx http.Context) {
    callbackgoravel.Handle(ctx, client, func(cb *duitku.CallbackRequest) error {
        // update order status based on cb.ResultCode
        return nil
    })
}
```

## API

### Client

| Method                                                     | Description                                 |
| ---------------------------------------------------------- | ------------------------------------------- |
| `NewClient(merchantCode, apiKey string, opts ...Option)` | Create new client                           |
| `CreateInvoice(ctx, req)`                                | Create payment invoice                      |
| `VerifyCallback(req)`                                    | Verify callback signature                   |
| `ProcessCallback(form, handler)`                         | Parse, verify & handle callback in one call |

### Options

| Option                     | Description            |
| -------------------------- | ---------------------- |
| `WithBaseURL(url)`       | Set API endpoint URL   |
| `WithHTTPClient(client)` | Set custom HTTP client |

### Constants

| Constant              | Value                              |
| --------------------- | ---------------------------------- |
| `BaseURLSandbox`    | `https://api-sandbox.duitku.com` |
| `BaseURLProduction` | `https://api-prod.duitku.com`    |

### Standalone Functions

| Function                                                                              | Description                                   |
| ------------------------------------------------------------------------------------- | --------------------------------------------- |
| `ParseCallback(form)`                                                               | Parse `url.Values` into `CallbackRequest` |
| `VerifyCallbackSignature(merchantCode, amount, merchantOrderID, apiKey, signature)` | Verify callback HMAC                          |
| `ParseCallbackResult(code)`                                                         | Convert result code to enum                   |

## Error Handling

`CreateInvoice` returns `*InvoiceError` on non-200 responses:

```go
var ie *duitku.InvoiceError
if errors.As(err, &ie) {
    log.Printf("HTTP %d: %s", ie.HTTPCode, ie.Body)
}
```

## Development

```bash
go test ./... -v
```
