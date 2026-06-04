package callbackecho

import (
	"github.com/labstack/echo/v4"
	"github.com/rachmanzz/duitku-client"
)

func Handle(c echo.Context, client *duitku.Client, handler duitku.CallbackHandler) error {
	c.Request().ParseForm()
	err := client.ProcessCallback(c.Request().PostForm, handler)
	if err != nil {
		return c.String(400, err.Error())
	}
	return c.String(200, "OK")
}
