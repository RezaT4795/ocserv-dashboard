package gateway

import "github.com/labstack/echo/v4"

func Routes(e *echo.Group) {
	ctl := New()

	g := e.Group("/gateway", tokenMiddleware())

	g.POST("/users", ctl.CreateUser)
	g.GET("/users/:username/status", ctl.UserStatus)
	g.PATCH("/users/:username/subscription", ctl.UpdateUserSubscription)
	g.DELETE("/users/:username", ctl.DeleteUser)
}
