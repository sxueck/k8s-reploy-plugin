package offline

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"log"
	"reflect"
	"strings"
)

func PostChangesMadeAfterSubmissionForCluster[
	T *appsv1.Deployment | *appsv1.StatefulSet](
	kubeClient *kubernetes.Clientset, namespace string, res T) error {

	var err error
	rt := reflect.ValueOf(res).Interface()
	switch rt.(type) {
	case *appsv1.StatefulSet:
		// is statefulset
		_, err = kubeClient.
			AppsV1().
			StatefulSets(namespace).
			Update(context.Background(), rt.(*appsv1.StatefulSet), v1.UpdateOptions{})
	case *appsv1.Deployment:
		// is deployment
		_, err = kubeClient.
			AppsV1().
			Deployments(namespace).
			Update(context.Background(), rt.(*appsv1.Deployment), v1.UpdateOptions{})
	}
	return err
}

func ErrReflectNotSlice() {
	log.Fatalln("unhealthy fields will be read by reflection, " +
		"it could be an exception in the resource fetch process")
}

func ChainChange(link string, v reflect.Value) reflect.Value {
	if !v.IsValid() {
		log.Fatalf("不存在 %s 字段，反射失败", link)
	}
	if strings.Count(link, ".") == 0 {
		if len(link) == 0 {
			log.Fatalln("wrong reflex chain")
		}
		return v.FieldByName(link)
	}
	k := link[:strings.Index(link, ".")]
	return ChainChange(link[len(k)+1:], v.FieldByName(k))
}

func changeImagePullPolicyPre(rv reflect.Value) (reflect.Value, reflect.Value) {
	endLinkV := ChainChange(ImagePullPolicyLink, rv)
	if endLinkV.Type().Kind() != reflect.Slice {
		ErrReflectNotSlice()
	}

	updatedMark := ChainChange(UpdatedAnnotations, rv)
	if updatedMark.Type().Kind() != reflect.Map {
		ErrReflectNotSlice()
	}
	return endLinkV, updatedMark
}

func changeImagesAction(rv reflect.Value, always bool, f func(bool) string) {
	endLinkV, updatedMark := changeImagePullPolicyPre(rv)
	for i := 0; i < endLinkV.Len(); i++ {
		policy := endLinkV.Index(i).FieldByName("ImagePullPolicy")
		if policy.String() == f(always) && policy.CanSet() {
			policy.SetString(f(!always))
			if len(updatedMark.MapKeys()) == 0 {
				log.Println("The resource label is empty and the identity cannot be injected correctly")
				return
			}
			updatedMark.SetMapIndex(
				reflect.ValueOf(UpdateMarkKey), reflect.ValueOf("TRUE"))
		}
	}
}

func reAlways(always bool) string {
	if always {
		return "Always"
	}
	return "IfNotPresent"
}

func WithChangeImagesPullPolicyToPresent(rv reflect.Value) {
	changeImagesAction(rv, false, reAlways)
}

func WithChangeImagesPullPolicyToAlways(rv reflect.Value) {
	changeImagesAction(rv, true, reAlways)
}
