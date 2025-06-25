package test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/relaxyabc/k8s-helper/crypto"
)

func TestAESEncryptBase64(t *testing.T) {
	cases := []map[string]string{
		{"name": "admin", "role": "admin"},
		{"name": "user", "role": "user"},
		{"name": "guest", "role": "guest"},
	}
	key := "k8s-mcp-client"
	for _, obj := range cases {
		plain, _ := json.Marshal(obj)
		sid, err := crypto.AESEncryptBase64(string(plain), key)
		if err != nil {
			t.Fatalf("encrypt error: %v", err)
		}
		fmt.Printf("加密后sid: %s | 用户: %s | 角色: %s\n", sid, obj["name"], obj["role"])
		plain2, err := crypto.AESDecryptBase64(sid, key)
		if err != nil {
			t.Fatalf("decrypt error: %v", err)
		}
		fmt.Printf("解密后json: %s | 用户: %s | 角色: %s\n", plain2, obj["name"], obj["role"])
	}
}
