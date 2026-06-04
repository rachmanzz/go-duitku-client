package callbackgoravel

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/rachmanzz/go-duitku-client"
)

func Handle(ctx http.Context, client *duitku.Client, handler duitku.CallbackHandler) {
	var data map[string]any
	if err := ctx.Request().Bind(&data); err != nil {
		ctx.Response().String(400, "duitku: bind form: "+err.Error())
		return
	}
	cb := duitku.ParseCallbackFromMap(data)
	if !client.VerifyCallback(cb) {
		ctx.Response().String(400, duitku.ErrInvalidSignature.Error())
		return
	}
	if err := handler(cb); err != nil {
		ctx.Response().String(400, err.Error())
		return
	}
	ctx.Response().String(200, "OK")
}
