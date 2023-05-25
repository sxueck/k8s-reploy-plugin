# Kuberneteus Redeployment Plugin

这是一个使用 Client-go 编写的插件，通过 WebHook 的方式，能够及时触发指定 Deployment 的 Images 修改和重部署

This is a plugin written using Client-go that can trigger the timely modification and redeployment of specified Deployments' images through WebHook.

## How to Redeployment a Service

Example:
```shell
## Request
curl -X "POST" "DOMAIN/webhook" \
     -H 'Content-Type: application/json; charset=utf-8' \
     -d $'{
  "namespace": "kube-public",
  "deployment": "k3s-reploy-webhook",
  "images": "registry.cn-shenzhen.aliyuncs.com/sxueck/k3s-webhook",
  "tag": "v1.1",
  "replicas": 1,
  "access-token": "token-xxx",
  "containers": "k3s-reploy-webhook"
}'
```

* images: the image name
* access-token: can be extracted from the environment variable `ACCESS_TOKEN`
* namespace: the namespace of the deployment
* replicas: the number of replicas, default 1
* tag: image tag
* resource: resource name, it will automatically determine whether it is `Deployment` or `StatefulSet`
* containers (optional): prevent multiple image in one pod, image name, if this is empty, it is automatically the same as the resource name
