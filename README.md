# duitku-client

Go SDK for [Duitku POP](https://docs.duitku.com/pop/en/) payment gateway — server-side integration.

## Install

Core (zero dependencies):

```bash
go get github.com/rachmanzz/go-duitku-client
```

Adapter framework (optional — pilih sesuai framework):

```bash
go get github.com/rachmanzz/go-duitku-client/callbackgin
# atau
go get github.com/rachmanzz/go-duitku-client/callbackecho
# atau
go get github.com/rachmanzz/go-duitku-client/callbackfiber
# atau
go get github.com/rachmanzz/go-duitku-client/callbackgoravel
```

> Setiap adapter adalah Go module terpisah — dependency framework cuma kepasang kalau beneran pake adapter itu.

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

Semua callback dari Duitku dikirim sebagai POST form. Library ini nyediain 3 cara handle:

1. Manual — `ProcessCallback(r.PostForm, handler)` — bebas pake framework apapun
2. Via `http.Handler` — `client.CallbackHandler(handler)` — tinggal mount di router
3. Adapter khusus framework

Pilih aja sesuai framework yang dipake.

#### net/http (standard library)

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

Atau pakai `CallbackHandler`:

```go
http.Handle("/callback", c.CallbackHandler(func(cb *duitku.CallbackRequest) error {
    // handle callback
    return nil
}))
```

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

Atau via `gin.WrapH`:

```go
r.POST("/callback", gin.WrapH(c.CallbackHandler(func(cb *duitku.CallbackRequest) error {
    // handle callback
    return nil
})))
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

Atau via `echo.WrapHandler`:

```go
e.POST("/callback", echo.WrapHandler(c.CallbackHandler(func(cb *duitku.CallbackRequest) error {
    // handle callback
    return nil
})))
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

Atau via `fiber.WrapH` / `adaptor.HTTPHandler`:

```go
import "github.com/gofiber/adaptor/v2"

app.Post("/callback", adaptor.HTTPHandler(c.CallbackHandler(func(cb *duitku.CallbackRequest) error {
    // handle callback
    return nil
})))
```

#### Goravel

Pake `Bind(&data)` idiomatic:

```go
import "github.com/rachmanzz/duitku-client/callbackgoravel"

func (c *Controller) Callback(ctx http.Context) {
    callbackgoravel.Handle(ctx, client, func(cb *duitku.CallbackRequest) error {
        // update order status based on cb.ResultCode
        return nil
    })
}
```

Atau manual:

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

## API

### Client

| Method                                                     | Description                                 |
| ---------------------------------------------------------- | ------------------------------------------- |
| `NewClient(merchantCode, apiKey string, opts ...Option)` | Create new client                           |
| `CreateInvoice(ctx, req)`                                | Create payment invoice                      |
| `VerifyCallback(req)`                                    | Verify callback signature                   |
| `ProcessCallback(form, handler)`                         | Parse, verify & handle callback in one call |
| `CallbackHandler(handler)`                               | Return `http.Handler` for direct mounting   |

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
