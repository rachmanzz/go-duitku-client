package callbackfiber

import (
	"net/url"

	"github.com/gofiber/fiber/v2"
	"github.com/rachmanzz/go-duitku-client"
)

func Handle(c *fiber.Ctx, client *duitku.Client, handler duitku.CallbackHandler) error {
	form, err := url.ParseQuery(string(c.Body()))
	if err != nil {
		return c.Status(400).SendString("duitku: parse form: " + err.Error())
	}
	err = client.ProcessCallback(form, handler)
	if err != nil {
		return c.Status(400).SendString(err.Error())
	}
	return c.SendString("OK")
}
