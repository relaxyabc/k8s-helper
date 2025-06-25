package test

import (
	"testing"

	"github.com/relaxyabc/k8s-helper/tools"
)

func TestListNamespacesByKubeConfigWithProxyAndInsecure(t *testing.T) {
	clientset, err := tools.GetK8sClient(kubeconfig, proxyAddr, insecure)
	if err != nil {
		t.Fatalf("获取 k8s client 失败: %v", err)
	}
	nsList, err := tools.ListNamespaces(clientset)
	if err != nil {
		t.Fatalf("获取 namespace 失败: %v", err)
	}
	if len(nsList) == 0 {
		t.Error("未获取到任何 namespace")
	}
	for _, ns := range nsList {
		t.Logf("namespace: %s", ns)
	}
}
