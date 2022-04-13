package main

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"log"
	"net/http"
	"os"
	"time"
)

type KMAP map[string]string

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
	startServ()
}

func startServ() {
	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	e.GET("/heathz", func(c echo.Context) error {
		return c.String(http.StatusOK, "working")
	})

	e.POST("/webhook", ReDeployWebhook)

	err := e.Start(":80")
	if err != nil {
		log.Printf("server error : %s", err)
		return
	}
}

func ReDeployWebhook(c echo.Context) error {
	var reCall = &ReCallDeployInfo{}
	err := c.Bind(&reCall)

	if err != nil {
		return c.String(http.StatusForbidden,
			fmt.Sprintf("bad format , %s", err))
	}

	if reCall.AccessToken != os.Getenv("WEBHOOK_TOKEN") {
		return c.String(http.StatusForbidden, "TOKEN ERROR")
	}

	if len(reCall.Containers) == 0 {
		reCall.Containers = reCall.Resource
	}

	var t = v1.Deployment{}
	containers := t.Spec.Template.Spec.Containers

	isDeployment := true

	kubeClient, err := NewInClusterClient()
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	deployment, err := kubeClient.
		AppsV1().
		Deployments(reCall.Namespace).
		Get(context.Background(), reCall.Resource, metav1.GetOptions{})

	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	var sts = &v1.StatefulSet{}
	if errors.IsNotFound(err) {
		sts, err = kubeClient.
			AppsV1().
			StatefulSets(reCall.Namespace).
			Get(context.Background(), reCall.Resource, metav1.GetOptions{})
		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}

		if errors.IsNotFound(err) {
			return c.String(http.StatusOK,
				fmt.Sprintf("resource not found : %s", err))
		}

		containers = sts.Spec.Template.Spec.Containers
		isDeployment = !isDeployment
	} else {
		containers = deployment.Spec.Template.Spec.Containers
	}

	found := false

	fmt.Printf("%+v\n", containers)

	for i := range containers {
		c := containers
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
			fmt.Sprintf("The application container not exist in the pod list"))
	}

	var deployERR error
	const annotationsKey = "redeploy.kubernetes.io/restartedAt"
	if isDeployment {

		// 通过更新 Annotations 时间戳，即使两个 Tag 相同，也能触发更新
		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			deployment.Annotations[annotationsKey] = time.Now().String()
			_, getErr := kubeClient.
				AppsV1().
				Deployments(reCall.Namespace).
				Update(context.Background(), deployment, metav1.UpdateOptions{})
			return getErr
		})
	} else {
		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			sts.Annotations[annotationsKey] = time.Now().String()
			_, getErr := kubeClient.
				AppsV1().
				StatefulSets(reCall.Namespace).
				Update(context.Background(), sts, metav1.UpdateOptions{})
			return getErr
		})
	}

	if deployERR != nil {
		return c.String(http.StatusOK, deployERR.Error())
	}

	return c.String(http.StatusOK, "successful")
}
