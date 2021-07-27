package main

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"os"
)

type ReCallDeployInfo struct {
	Namespace  string `json:"namespace"`
	Deployment string `json:"deployment"`
	Images     string `json:"images"`
	Tag        string `json:"tag"`
	Replicas   int    `json:"replicas"`
	Containers string `json:"containers"`

	AccessToken string `json:"access-token"`
}

func main() {
	startServ()
}

func startServ() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())

	e.GET("/heathz", func(c echo.Context) error {
		return c.String(http.StatusOK, "working")
	})

	e.POST("/webhook", ReDeployWebhook)

	err := e.Start(":80")
	if err != nil {
		return
	}
}

func ReDeployWebhook(c echo.Context) error {
	var reCall = &ReCallDeployInfo{}
	err := c.Bind(&reCall)

	if err != nil {
		return c.String(http.StatusForbidden, fmt.Sprintf("bad format , %s", err))
	}

	if reCall.AccessToken != os.Getenv("WEBHOOK_TOKEN") {
		return c.String(http.StatusForbidden, "token error")
	}

	//fmt.Printf("%+v\n", reCall)

	kubeClient, err := NewInClusterClient()
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	depolyment, err := kubeClient.AppsV1().Deployments(reCall.Namespace).Get(
		context.Background(), reCall.Deployment, metav1.GetOptions{})
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}
	if errors.IsNotFound(err) {
		return c.String(http.StatusOK, "Deployment not found")
	}

	containers := &depolyment.Spec.Template.Spec.Containers
	found := false

	fmt.Printf("%+v\n", containers)

	for i := range *containers {
		c := *containers
		if c[i].Name == reCall.Containers {
			found = true
			newImages := fmt.Sprintf("%s:%s", reCall.Images, reCall.Tag)
			log.Println("old images =>", c[i].Image)
			log.Println("new images =>", newImages)
			c[i].Image = newImages
		}
	}

	if found == false {
		return c.String(http.StatusOK,
			fmt.Sprintf("The application container not exist in the deployment pods.\n"))
	}

	_, err = kubeClient.AppsV1().Deployments(reCall.Namespace).Update(
		context.Background(), depolyment, metav1.UpdateOptions{})
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	return c.String(http.StatusOK, "successful")
}
