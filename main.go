package main

import (
	"context"
	"fmt"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	rconfig "github.com/sxueck/redeploy/config"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
	"log"
	"net/http"
	"time"
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

	if reCall.AccessToken != rconfig.Cfg.WebhookToken {
		return c.String(http.StatusForbidden, "TOKEN ERROR")
	}

	if len(reCall.Containers) == 0 {
		reCall.Containers = reCall.Resource
	}

	var containers []corev1.Container

	isDeployment := true

	kubeClient, err := NewInClusterClient()
	if err != nil {
		return c.String(http.StatusOK, err.Error())
	}

	deployment, err := kubeClient.
		AppsV1().
		Deployments(reCall.Namespace).
		Get(context.Background(), reCall.Resource, metav1.GetOptions{})

	var statefulSet *v1.StatefulSet
	if errors.IsNotFound(err) {
		log.Printf("%s statfulset", reCall.Resource)
		var getErr error
		statefulSet, getErr = kubeClient.
			AppsV1().
			StatefulSets(reCall.Namespace).
			Get(context.Background(), reCall.Resource, metav1.GetOptions{})

		if getErr != nil {
			log.Println(getErr)
			return c.String(http.StatusOK, err.Error())
		}

		containers = statefulSet.Spec.Template.Spec.Containers
		isDeployment = !isDeployment
	} else {
		if err != nil {
			return c.String(http.StatusOK, err.Error())
		}
		containers = deployment.Spec.Template.Spec.Containers
	}

	fmt.Printf("%+v\n", containers)

	found := false
	for i, v := range containers {
		if v.Name == reCall.Containers {
			found = true
			newImages := fmt.Sprintf("%s:%s", reCall.Images, reCall.Tag)
			log.Println("old images =>", v.Image)
			log.Println("new images =>", newImages)

			containers[i].Image = newImages
		}
	}

	if !found {
		return c.String(http.StatusOK,
			fmt.Sprintf("The application container not exist in the pod list"))
	}

	var deployERR error
	const annotationsKey = "redeploy.kubernetes.io/restartedAt"
	if isDeployment {

		// 通过更新 Annotations 时间戳，即使两个 Tag 相同，也能触发更新
		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			deployment.
				Spec.
				Template.
				ObjectMeta.
				Annotations[annotationsKey] = time.Now().String()

			_, getErr := kubeClient.
				AppsV1().
				Deployments(reCall.Namespace).
				Update(context.Background(), deployment, metav1.UpdateOptions{})
			return getErr
		})

	} else {

		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			_, getErr := kubeClient.
				AppsV1().
				StatefulSets(reCall.Namespace).
				Update(context.Background(), statefulSet, metav1.UpdateOptions{})

			var replicas int32 = 0
			statefulSet.Spec.Replicas = &replicas
			// statefulSet 更新只能靠 scale
			_, scaleErr := kubeClient.AppsV1().StatefulSets(reCall.Namespace).
				Update(context.Background(), statefulSet, metav1.UpdateOptions{})

			if scaleErr != nil {
				return scaleErr
			}

			return getErr
		})

		go func() {
			replicas := int32(reCall.Replicas)

			for i := 0; i <= 100; i++ {
				replicaCount, err := kubeClient.AppsV1().StatefulSets(reCall.Namespace).
					GetScale(context.Background(), reCall.Resource, metav1.GetOptions{})
				if err != nil {
					log.Println(err)
					return
				}

				if replicaCount.Spec.Replicas == 0 {
					break
				}

				<-time.NewTicker(5 * time.Second).C
				log.Println("waiting...")
			}

			statefulSet.Spec.Replicas = &replicas
			_, sErr := kubeClient.AppsV1().StatefulSets(reCall.Namespace).
				Update(context.Background(), statefulSet, metav1.UpdateOptions{})
			if sErr != nil {
				log.Printf("scale recovery of anomalies :%s", sErr)
			}
		}()
	}

	if deployERR != nil {
		return c.String(http.StatusOK, deployERR.Error())
	}

	return c.String(http.StatusOK, "successful")
}
