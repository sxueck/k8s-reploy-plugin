package model

type ReCallDeployInfo struct {
	Namespace  string `json:"namespace"`
	Resource   string `json:"resource"`
	Images     string `json:"images"`
	Tag        string `json:"tag"`
	Replicas   int    `json:"replicas"`
	Containers string `json:"containers"`

	AccessToken string `json:"access-token"`
	Debug       bool   `json:"debug"`
}
