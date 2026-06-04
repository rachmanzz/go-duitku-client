package callbackgin

import (
	"github.com/gin-gonic/gin"
	"github.com/rachmanzz/duitku-client"
)

func Handle(c *gin.Context, client *duitku.Client, handler duitku.CallbackHandler) {
	c.Request.ParseForm()
	err := client.ProcessCallback(c.Request.PostForm, handler)
	if err != nil {
		c.String(400, err.Error())
		return
	}
	c.String(200, "OK")
}
