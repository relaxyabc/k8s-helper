package test

import (
	"fmt"
	"testing"

	"github.com/relaxyabc/k8s-helper/tools"
)

var kubeconfig = `{"apiVersion":"v1","clusters":[{"cluster":{"certificate-authority-data":null,"server":"https://127.0.0.1:6443","insecure-skip-tls-verify":true},"name":"kubernetes"}],"contexts":[{"context":{"cluster":"kubernetes","user":"kubernetes-admin"},"name":"kubernetes-admin@kubernetes"}],"current-context":"kubernetes-admin@kubernetes","kind":"Config","preferences":{},"users":[{"name":"kubernetes-admin","user":{"client-certificate-data":"LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSURJVENDQWdtZ0F3SUJBZ0lJY1NSL1AvOG55TVV3RFFZSktvWklodmNOQVFFTEJRQXdGVEVUTUJFR0ExVUUKQXhNS2EzVmlaWEp1WlhSbGN6QWVGdzB5TlRBMU16QXdNak13TURoYUZ3MHlOakExTXpBd01qTTFNRGxhTURReApGekFWQmdOVkJBb1REbk41YzNSbGJUcHRZWE4wWlhKek1Sa3dGd1lEVlFRREV4QnJkV0psY201bGRHVnpMV0ZrCmJXbHVNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DQVE4QU1JSUJDZ0tDQVFFQTJzSGNhWGlveVJ4UVJCS1QKRjh0ZjRVb20zSlM3Nm5XSjNLdmtoV3NkUWJ0eUtVN2ZzcjNvMXg0cUZTMzNUeVcxTy90UXdrSTdjOWVDYmp6MQpjZmdrTFFrZkFtREFqb2dpcExoVDJ3Q0MyNjVSd09sQ25kL3ZJUCsxZ3k4Y1RrR2wvcVY0V0pjRjArdjV3UmpWCmNPZFJGMjNZakxIakQrQVVibnNPS095cXpIdmhhaWxjRjIwelVCdG5CNlQzcFVUdE1GSzVsdVJvRDhDVFhKcXcKZFo3QXc1dGJ4MjB5NHltZTNPZlB4K096aEQ4c2FWbHFlT2FvSTZNRHZsRmFYVzZaZ0dJV0JzbnI1S1V5eitEdQpzMGdlc2NLOVl1SVAzRExEU2lmMDBJNXBQeWNWNG9rOFl3OFA0MVE0OUlDV0EzcXQxMm9SYlZMN1FUSVp2WURjCmFURzUvd0lEQVFBQm8xWXdWREFPQmdOVkhROEJBZjhFQkFNQ0JhQXdFd1lEVlIwbEJBd3dDZ1lJS3dZQkJRVUgKQXdJd0RBWURWUjBUQVFIL0JBSXdBREFmQmdOVkhTTUVHREFXZ0JSemVld3FENFhZTExKdTJSK0FvVW53N3FCQwpYREFOQmdrcWhraUc5dzBCQVFzRkFBT0NBUUVBQVA2b09pS2JhM1V5aHRFeUl3eGZrSEUwdnluZThHSm5CTURQCmZKNzZIUGpxZVRvMTJhc1NEUUJHdUNYelBHREtuL2thRE5rZnJvNlVMQmFFNkFucWFjc3doQ2RYQldqaTJZSmsKWmdqWkh3L2pTSjIrNUZ6b1BNK1RyeUU4UWp5L2p6eGpqenFFUnpVRVRhNmNKVkplclJZVXBFbHNVVFBRc25WOAppS2ZJRjA5eVFZcFJJRkludkdBVy9JKzFObmVzRlkyZEI3ZC8wZWd5THRVdE9OYTZJejZYcSs0UWZEYUFueWNECm8xSVpjbWd1bHZETGc0enYvaXRPRktvekVlUUg0S3JJZFovTTZlcnBVRGhmdUozMGkwYlF0RHgzUUorL3hybGYKek5wczZMTDZBQkF6UExyKytVdEFKMUNHbEtsc1hRY1dJQkxmQzd0M1p3Y3grdmFXTmc9PQotLS0tLUVORCBDRVJUSUZJQ0FURS0tLS0tCg==","client-key-data":"LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFb2dJQkFBS0NBUUVBMnNIY2FYaW95UnhRUkJLVEY4dGY0VW9tM0pTNzZuV0ozS3ZraFdzZFFidHlLVTdmCnNyM28xeDRxRlMzM1R5VzFPL3RRd2tJN2M5ZUNianoxY2Zna0xRa2ZBbURBam9naXBMaFQyd0NDMjY1UndPbEMKbmQvdklQKzFneThjVGtHbC9xVjRXSmNGMCt2NXdSalZjT2RSRjIzWWpMSGpEK0FVYm5zT0tPeXF6SHZoYWlsYwpGMjB6VUJ0bkI2VDNwVVR0TUZLNWx1Um9EOENUWEpxd2RaN0F3NXRieDIweTR5bWUzT2ZQeCtPemhEOHNhVmxxCmVPYW9JNk1EdmxGYVhXNlpnR0lXQnNucjVLVXl6K0R1czBnZXNjSzlZdUlQM0RMRFNpZjAwSTVwUHljVjRvazgKWXc4UDQxUTQ5SUNXQTNxdDEyb1JiVkw3UVRJWnZZRGNhVEc1L3dJREFRQUJBb0lCQUN5a0x4U2UrT0xCa21leAoycTZhWkNDWGYwSzRYM2pETDdVR3B3WExEQmRqNXpJaTFHZW5kYUtnbHpublBYYTdYVGEyWEk2bjhYWDhYck9jCllRSTIvenJwaDVoMm9oVDBGdzNDUitxRE9qRVdhN3lRWFhYV1F2aGE5bkdrNVlQYXhjTW5oVjJleENPeEhqQ1gKbnNjQmpYellmNzZHZHEzQXJxcTFGZmlvQTFyaTlETyt4TEZreVVNZ3FRaXdPcW54UWo5eTlQMXlFZkFnNGpqaApsbGxrRzVuQ2l5YWtFUFZXTUQ1ZHBpR3c2ZXBMOFFidTVLRWVjZVZRdnZxUVQ0SVBaWkk5VXlmYlRRTXd1TUNsCkhocEdXUktwVllKTkFWTFZFVk5IeCs4T0pKTGtwVzJXNE5UdHp5dHhsZytSdTRPMExXWmROR3pub0xSNHArcWUKZmlnWU1CRUNnWUVBK0gvQ2RwcllsQkhUaDBLZVVpZkRRT1VXV3VFeUZ5cUtHUzVFTkZWMHdLRjltRUtxQW1jdgowemNoaDNLT0wzVWRjT2cxeFpSZWFBMmRsdE9ZMGM4aG9NZSsraEgwbkxHL3MrVk9nK2NMVU9jVTcyeWdYSkZYCmpmRFdkaHdYNnQ4Zy9vSCtOZFcySlpBMkxGejJNenhhMGR6M0s5dDFBbXhENkZEcFNuWC9tWnNDZ1lFQTRWeEcKcDAvQ3gzRTh1d2U2dmtHWVFhbFo2b3o3TGtpVWtoek95Umd4UVY3VXFlSnJFYTErQ0trUnB4alRseEFOcFkwcApaUnpUaUxiQ25aYU1zcDgrS2JzUjgyUk9KeVVvYW53ME1CVThLbUVwbS9pVkF3b2NvYVZBU0ZPa29jNDN4dHpyCllGNUk4d2QrVjB4NG1nZ0ZPemQxOGIza1M5UHlndnM2UWs5Q3FXMENnWUEzMG10YXZWb2RtUXVOZlBlWHVQcngKbndTd2tabncva3RiY0xzOWpselYwUEVudlFIMzNEb3dGbGhmMXVuOTJ5OHI1OTM4Ym1IdXVmQkdxMjNPNDlySQpCVmJ2VWcxRERlTGtoSVJvdVFRZnZtbERoNEZXaWdmRENQRUVRemRVT1o5dHpNSFFVaHZDd0d5SzlxOFB2MlVmCmM2WEtvbGZjblhsN3ZJRkxpc3BLTlFLQmdHbStJWXphR1J2Nkh6UG5FWkc2TjVYL3Y2Z1YxTHBINWlhVkM5WkIKMnNMQW0yckhTZFAyb3grdkxSQkp6dWFmNnJkV2dDam9tTDBhZkVEelpqdGVDdzRMc0FXVGVEUlg5Qm5iQTZYWQpJTzRGdno5bktZeE9qMWF0c25iOWdFOUg4dFlGelEvZnpienpOQzRFUE1hUm90ckJVRDlKQ2JrbXp6RDBic2EwCmFDUVZBb0dBTFlncXVJM3QzbWZsczFtd0p5VTB4UjdLUkFGbTdSdVl5MVVuKzRSN3M3bS84b2xqQUg5T3VEdHUKWmFHVjhWV25RVlZhMUZNOUFBcUVxTmJXR1Y1WDZPVlRoaE1ibllDcVpXUkMvcE5rcjF6cUwraitYdXlYeE9TQwoyL3NwdkJTQVZyYUNBRE1hVlRmRFg4QzZubDN0SDg2OS9NMzFTKzh3eFVOZWJCZldRdU09Ci0tLS0tRU5EIFJTQSBQUklWQVRFIEtFWS0tLS0tCg=="}}]}`
var proxyAddr = ""
var insecure = true

func TestListPods(t *testing.T) {
	namespace := "default"
	clientset, err := tools.GetK8sClient(kubeconfig, proxyAddr, insecure)
	if err != nil {
		t.Fatalf("获取 k8s client 失败: %v", err)
	}
	pods, err := tools.ListPods(clientset, namespace)
	if err != nil {
		t.Fatalf("获取 pods 失败: %v", err)
	}
	for _, pod := range pods {
		fmt.Printf("[POD] %s\n", pod)
	}
}

func TestListDeployments(t *testing.T) {
	namespace := "default"
	clientset, err := tools.GetK8sClient(kubeconfig, proxyAddr, insecure)
	if err != nil {
		t.Fatalf("获取 k8s client 失败: %v", err)
	}
	deployments, err := tools.ListDeployments(clientset, namespace)
	if err != nil {
		t.Fatalf("获取 deployments 失败: %v", err)
	}
	for _, deploy := range deployments {
		fmt.Printf("[DEPLOYMENT] %s\n", deploy)
	}
}

func TestListDaemonSets(t *testing.T) {
	namespace := "default"
	clientset, err := tools.GetK8sClient(kubeconfig, proxyAddr, insecure)
	if err != nil {
		t.Fatalf("获取 k8s client 失败: %v", err)
	}
	daemonsets, err := tools.ListDaemonSets(clientset, namespace)
	if err != nil {
		t.Fatalf("获取 daemonsets 失败: %v", err)
	}
	for _, ds := range daemonsets {
		fmt.Printf("[DAEMONSET] %s\n", ds)
	}
}
