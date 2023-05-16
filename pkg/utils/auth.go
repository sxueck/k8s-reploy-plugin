package utils

import "github.com/labstack/echo/v4"

const(
	AuthHeader = "X-Authentication"
)

func ConnectMiddlewareAuth(c echo.Context) bool {
	return true
}
