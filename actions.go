package main

import (
	"context"
	"fmt"
	"github.com/golang-module/carbon/v2"
	"github.com/labstack/echo/v4"
	rconfig "github.com/sxueck/k8sodep/config"
	"github.com/sxueck/k8sodep/model"
	"github.com/sxueck/k8sodep/offline"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/retry"
	"log"
	"net/http"
)

var cfg = rconfig.Cfg

func NewInClusterClient() (*kubernetes.Clientset, error) {
	var (
		config *rest.Config
		err    error
	)

	apiServer := cfg.ApiServer
	if len(apiServer) > 0 {

		log.Printf("apiServer Address is : %s", apiServer)

		if cfg.KubeConfig == "" {
			return nil, fmt.Errorf("please configure KUBECONFIG parameters")
		}

		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeConfig)
		if err != nil {
			return nil, err
		}

	} else {
		//return nil, fmt.Errorf("please define API_SERVER parameter to specify k8s address")
		config, err = rest.InClusterConfig()
		if err != nil {
			log.Printf("failed to obtain config from InClusterConfig: %v", err)
			return nil, err
		}
	}

	return kubernetes.NewForConfig(config)
}

func ReDeployWebhook(c echo.Context) error {
	var reCall = &model.ReCallDeployInfo{}
	err := c.Bind(&reCall)
	log.Printf("ReCall : %+v\n", reCall)

	if err != nil {
		return c.String(http.StatusForbidden,
			fmt.Sprintf("bad format, %s", err))
	}

	if reCall.AccessToken != rconfig.Cfg.WebhookToken {
		return c.String(http.StatusForbidden, "TOKEN ERROR")
	}

	err = ExecuteRedeployment(*reCall, false)
	if err != nil {
		return c.String(http.StatusInternalServerError,
			fmt.Sprintf("redeployment failed, %s", err))
	}

	return c.String(http.StatusOK, "successful")
}

func ExecuteRedeployment(reCall model.ReCallDeployInfo, restore bool) error {
	var now = carbon.Now().ToDateTimeString() // 2020-08-05 13:14:15

	if len(reCall.Containers) == 0 {
		reCall.Containers = reCall.Resource
	}

	var containers []corev1.Container
	isDeployment := true

	kubeClient, err := NewInClusterClient()
	if err != nil {
		return err
	}

	deployment, err := kubeClient.
		AppsV1().
		Deployments(reCall.Namespace).
		Get(context.Background(), reCall.Resource, metav1.GetOptions{})
	log.Println(deployment)

	var statefulSet *appsv1.StatefulSet
	if errors.IsNotFound(err) {
		log.Printf("%s statfulset", reCall.Resource)
		var getErr error
		statefulSet, getErr = kubeClient.
			AppsV1().
			StatefulSets(reCall.Namespace).
			Get(context.Background(), reCall.Resource, metav1.GetOptions{})

		if getErr != nil {
			return getErr
		}

		containers = statefulSet.Spec.Template.Spec.Containers
		isDeployment = !isDeployment
	} else {
		if err != nil {
			return err
		}
		containers = deployment.Spec.Template.Spec.Containers
	}

	fmt.Printf("%+v\n", containers)

	nowPullPolicy := ""
	containedIndex := -1
	found := false
	for i, v := range containers {
		if v.Name == reCall.Containers {
			found = true
			newImages := fmt.Sprintf("%s:%s", reCall.Images, reCall.Tag)
			nowPullPolicy = string(v.ImagePullPolicy)
			log.Println("old images =>", v.Image)
			log.Println("new images =>", newImages)

			containers[i].Image = newImages
			nowPullPolicy = string(v.ImagePullPolicy)
			containedIndex = i
			break
		}
	}

	if !found || containedIndex == -1 {
		return fmt.Errorf("the application container not exist in the pod list")
	}

	var deployERR error
	const annotationsKey = "redeploy.kubernetes.io/restartedAt"
	var annotationsNotFound = fmt.Errorf("未找到 Spec.Template.Annotations 字段，请检查资源状态")
	if isDeployment {
		// 注意apiserver到etcd的链路不为原子操作，且大部分为乐观锁，可能会遇到资源conflict的情况

		// 通过更新 Annotations 时间戳，即使两个 Tag 相同，也能触发更新
		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			annotations := &deployment.
				Spec.
				Template.
				ObjectMeta.
				Annotations

			if *annotations != nil {
				(*annotations)[annotationsKey] = now
			} else {
				return annotationsNotFound
			}

			if reCall.LinkCloud {
				containers[containedIndex].ImagePullPolicy = corev1.PullAlways
			}

			if restore {
				containers[containedIndex].ImagePullPolicy = corev1.PullIfNotPresent
			}

			return offline.PostChangesMadeAfterSubmissionForCluster(kubeClient, reCall.Namespace, deployment)
		})

	} else {
		deployERR = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			annotations := &statefulSet.
				Spec.
				Template.
				ObjectMeta.
				Annotations

			if *annotations != nil {
				(*annotations)[annotationsKey] = now
			} else {
				return annotationsNotFound
			}

			return offline.PostChangesMadeAfterSubmissionForCluster(kubeClient, reCall.Namespace, statefulSet)
		})
	}

	// 更新完成恢复成原来的状态
	if !restore && isDeployment && corev1.PullPolicy(nowPullPolicy) == corev1.PullIfNotPresent {
		reCall.LinkCloud = false
		return ExecuteRedeployment(reCall, true)
	}

	return deployERR
}
