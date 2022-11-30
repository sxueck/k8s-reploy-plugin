package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sxueck/redeploy/bigger"
	"github.com/sxueck/redeploy/pkg/utils"
	"log"
	"net/http"
)

type ReCallDeployInfo struct {
	Namespace  string `json:"namespace"`
	Resource   string `json:"resource"`
	Images     string `json:"images"`
	Tag        string `json:"tag"`
	Replicas   int    `json:"replicas"`
	Containers string `json:"containers"`

	AccessToken string `json:"access-token"`
}

func main() {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	startServ()
}

func startServ() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())
	e.Use(middleware.BodyLimit("2048M"))

	e.GET("/heathz", func(c echo.Context) error {
		return c.String(http.StatusOK, "working")
	})

	i := e.Group("/images")
	i.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if !utils.ConnectMiddlewareAuth(c) {
				return c.String(http.StatusForbidden,"Authentication failed")
			}
			return next(c)
		}
	})
	// TODO: 支持分片上传，断点续传
	// 上传镜像文件的接口
	i.POST("/images/upload", bigger.ShareColumnImagesUploadHandler)
	// 分片沟通消息接口
	i.GET("/images/process", bigger.ImagesInfoHandler)
	e.POST("/webhook", ReDeployWebhook)

	err := e.Start(":80")
	if err != nil {
		log.Printf("server error : %s", err)
		return
	}
}