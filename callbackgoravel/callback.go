package callbackgoravel

import (
	"github.com/goravel/framework/contracts/http"
	"github.com/rachmanzz/go-duitku-client"
)

func Handle(ctx http.Context, client *duitku.Client, handler duitku.CallbackHandler) {
	r := ctx.Request().Origin()
	r.ParseForm()
	err := client.ProcessCallback(r.PostForm, handler)
	if err != nil {
		ctx.Response().String(400, err.Error())
		return
	}
	ctx.Response().String(200, "OK")
}
