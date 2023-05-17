package main

import (
	"github.com/labstack/echo/v4"
	"github.com/sxueck/k8sodep/bigger"
	"github.com/sxueck/k8sodep/pkg/utils"
	"log"
	"net/http"
)

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	startServ()
}

func startServ() {
	e := echo.New()
	e.HideBanner = true
	//e.Use(middleware.Recover())
	//e.Use(middleware.Logger())
	//e.Use(middleware.BodyLimit("1024M"))

	e.GET("/heathz", func(c echo.Context) error {
		return c.String(http.StatusOK, "working")
	})

	i := e.Group("/images")
	i.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !utils.ConnectMiddlewareAuth(c) {
				return c.String(http.StatusForbidden, "Authentication failed")
			}
			return next(c)
		}
	})

	// 上传镜像文件的接口
	i.POST("/register", bigger.RegisterUploadTaskToDaemon)
	i.POST("/upload", nil, bigger.StartRecvUploadHandle())

	e.POST("/webhook", ReDeployWebhook)

	err := e.Start(":80")
	if err != nil {
		log.Printf("server error : %s", err)
		return
	}
}
