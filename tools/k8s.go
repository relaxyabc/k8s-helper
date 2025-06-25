package tools

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/relaxyabc/k8s-helper/dao"
	"golang.org/x/net/proxy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// GetK8sClient 获取 k8s clientset，支持可选 socks5 代理和跳过 TLS 校验
func GetK8sClient(kubeconfigData string, proxyAddr string, insecure bool) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigData))
	if err != nil {
		return nil, fmt.Errorf("failed to build config from kubeconfig: %w", err)
	}
	if insecure {
		config.Insecure = true
		config.TLSClientConfig.Insecure = true
		config.TLSClientConfig.CAFile = ""
		config.TLSClientConfig.CAData = nil
	}
	if proxyAddr != "" {
		dialer, err := proxy.SOCKS5("tcp", proxyAddr, nil, proxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("failed to create SOCKS5 dialer: %w", err)
		}
		contextDialer, ok := dialer.(proxy.ContextDialer)
		if !ok {
			return nil, fmt.Errorf("dialer does not support context")
		}
		config.WrapTransport = func(rt http.RoundTripper) http.RoundTripper {
			transport, ok := rt.(*http.Transport)
			if !ok {
				return rt
			}
			clonedTransport := transport.Clone()
			clonedTransport.DialContext = contextDialer.DialContext
			return clonedTransport
		}
	}
	return kubernetes.NewForConfig(config)
}

// ListNamespaces 获取 namespace 列表
func ListNamespaces(clientset *kubernetes.Clientset) ([]string, error) {
	nsList, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, ns := range nsList.Items {
		result = append(result, ns.Name)
	}
	return result, nil
}

// ListPods 获取指定命名空间下的 Pod 名称列表
func ListPods(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, pod := range pods.Items {
		result = append(result, pod.Name)
	}
	return result, nil
}

// ListDeployments 获取指定命名空间下的 Deployment 名称列表
func ListDeployments(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	deployments, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, deploy := range deployments.Items {
		result = append(result, deploy.Name)
	}
	return result, nil
}

// ListDaemonSets 获取指定命名空间下的 DaemonSet 名称列表
func ListDaemonSets(clientset *kubernetes.Clientset, namespace string) ([]string, error) {
	daemonsets, err := clientset.AppsV1().DaemonSets(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var result []string
	for _, ds := range daemonsets.Items {
		result = append(result, ds.Name)
	}
	return result, nil
}

// RolloutRestartDeployment 滚动重启 Deployment
func RolloutRestartDeployment(clientset *kubernetes.Clientset, namespace, name string) error {
	deploymentsClient := clientset.AppsV1().Deployments(namespace)
	deployment, err := deploymentsClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = map[string]string{}
	}
	deployment.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")
	_, err = deploymentsClient.Update(context.Background(), deployment, metav1.UpdateOptions{})
	return err
}

// RolloutRestartDaemonSet 滚动重启 DaemonSet
func RolloutRestartDaemonSet(clientset *kubernetes.Clientset, namespace, name string) error {
	daemonsetsClient := clientset.AppsV1().DaemonSets(namespace)
	daemonset, err := daemonsetsClient.Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if daemonset.Spec.Template.Annotations == nil {
		daemonset.Spec.Template.Annotations = map[string]string{}
	}
	daemonset.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = metav1.Now().Format("2006-01-02T15:04:05Z07:00")
	_, err = daemonsetsClient.Update(context.Background(), daemonset, metav1.UpdateOptions{})
	return err
}

// RolloutRestartDeploymentTool 滚动重启 Deployment
func RolloutRestartDeploymentTool(proxy string, clusterName, namespace, name string) error {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return err
	}
	return RolloutRestartDeployment(clientset, namespace, name)
}

// RolloutRestartDaemonSetTool 滚动重启 DaemonSet
func RolloutRestartDaemonSetTool(proxy string, clusterName, namespace, name string) error {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return err
	}
	return RolloutRestartDaemonSet(clientset, namespace, name)
}

// GetNamespacesTool 查询指定集群的 namespace 列表
func GetNamespacesTool(proxy, clusterName string) ([]string, error) {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return nil, err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return nil, err
	}
	return ListNamespaces(clientset)
}

// GetPodsTool 获取指定集群和命名空间下的 Pod 名称列表
func GetPodsTool(proxy, clusterName, namespace string) ([]string, error) {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return nil, err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return nil, err
	}
	return ListPods(clientset, namespace)
}

// GetDeploymentsTool 获取指定集群和命名空间下的 Deployment 名称列表
func GetDeploymentsTool(proxy, clusterName, namespace string) ([]string, error) {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return nil, err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return nil, err
	}
	return ListDeployments(clientset, namespace)
}

// GetDaemonSetsTool 获取指定集群和命名空间下的 DaemonSet 名称列表
func GetDaemonSetsTool(proxy, clusterName, namespace string) ([]string, error) {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return nil, err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return nil, err
	}
	return ListDaemonSets(clientset, namespace)
}

// HTTPRequestTool 实现简单的 HTTP 请求
func HTTPRequestTool(method, url, body string) (string, error) {
	client := &http.Client{}
	var req *http.Request
	var err error
	if method == "POST" || method == "PUT" {
		req, err = http.NewRequest(method, url, strings.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(respBody), nil
}

// GetK8sVersionTool 获取指定集群的 k8s 版本
func GetK8sVersionTool(proxy, clusterName string) (string, error) {
	kubeconfig, err := dao.GetKubeConfig(clusterName)
	if err != nil {
		return "", err
	}
	clientset, err := GetK8sClient(kubeconfig, proxy, true)
	if err != nil {
		return "", err
	}
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return versionInfo.String(), nil
}
