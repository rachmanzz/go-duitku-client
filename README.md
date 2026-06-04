# duitku-client

Go SDK for [Duitku POP](https://docs.duitku.com/pop/en/) payment gateway — server-side integration.

## Install

```bash
go get github.com/rachmanzz/go-duitku-client
```

Zero dependencies.

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

Semua callback dari Duitku dikirim sebagai POST form. Cukup pake `ProcessCallback` — works with any framework:

#### net/http

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

#### Gin

```go
r.POST("/callback", func(c *gin.Context) {
    r := c.Request
    r.ParseForm()
    err := client.ProcessCallback(r.PostForm, func(cb *duitku.CallbackRequest) error {
        // handle callback
        return nil
    })
    if err != nil {
        c.String(400, err.Error())
        return
    }
    c.String(200, "OK")
})
```

#### Echo

```go
e.POST("/callback", func(c echo.Context) error {
    r := c.Request()
    r.ParseForm()
    err := client.ProcessCallback(r.PostForm, func(cb *duitku.CallbackRequest) error {
        // handle callback
        return nil
    })
    if err != nil {
        return c.String(400, err.Error())
    }
    return c.String(200, "OK")
})
```

#### Fiber

```go
import "net/url"

app.Post("/callback", func(c fiber.Ctx) error {
    form, err := url.ParseQuery(string(c.Body()))
    if err != nil {
        return c.Status(400).SendString("parse form: " + err.Error())
    }
    err = client.ProcessCallback(form, func(cb *duitku.CallbackRequest) error {
        // handle callback
        return nil
    })
    if err != nil {
        return c.Status(400).SendString(err.Error())
    }
    return c.SendString("OK")
})
```

#### Goravel

```go
func (c *Controller) Callback(ctx http.Context) {
    var data map[string]any
    ctx.Request().Bind(&data)
    cb := duitku.ParseCallbackFromMap(data)
    if !client.VerifyCallback(cb) {
        return ctx.Response().String(400, "invalid signature")
    }
    // handle callback
    return ctx.Response().String(200, "OK")
}
```

Atau via `Origin()`:

```go
func (c *Controller) Callback(ctx http.Context) {
    r := ctx.Request().Origin()
    r.ParseForm()
    err := client.ProcessCallback(r.PostForm, func(cb *duitku.CallbackRequest) error {
        // handle callback
        return nil
    })
    if err != nil {
        return ctx.Response().String(400, err.Error())
    }
    return ctx.Response().String(200, "OK")
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
| `ParseCallbackFromMap(data)`                                                        | Parse `map[string]any` into `CallbackRequest` |
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
