package crypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"errors"
)

// PKCS7 填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func pkcs7Unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("invalid padding size")
	}
	unpadding := int(data[length-1])
	if unpadding > length {
		return nil, errors.New("invalid padding size")
	}
	return data[:(length - unpadding)], nil
}

// AES CBC 加密并 base64
func AESEncryptBase64(plain, key string) (string, error) {
	k := []byte(key)
	if len(k) < 16 {
		k = append(k, bytes.Repeat([]byte("0"), 16-len(k))...)
	} else if len(k) > 16 {
		k = k[:16]
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	iv := k // IV 直接用 key（简单实现）
	pad := pkcs7Pad([]byte(plain), block.BlockSize())
	blockMode := cipher.NewCBCEncrypter(block, iv)
	crypted := make([]byte, len(pad))
	blockMode.CryptBlocks(crypted, pad)
	return base64.StdEncoding.EncodeToString(crypted), nil
}

// AES CBC 解密 base64
func AESDecryptBase64(cipherText, key string) (string, error) {
	k := []byte(key)
	if len(k) < 16 {
		k = append(k, bytes.Repeat([]byte("0"), 16-len(k))...)
	} else if len(k) > 16 {
		k = k[:16]
	}
	block, err := aes.NewCipher(k)
	if err != nil {
		return "", err
	}
	iv := k
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return "", err
	}
	if len(raw)%block.BlockSize() != 0 {
		return "", errors.New("cipherText is not a multiple of the block size")
	}
	blockMode := cipher.NewCBCDecrypter(block, iv)
	plain := make([]byte, len(raw))
	blockMode.CryptBlocks(plain, raw)
	plain, err = pkcs7Unpad(plain)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}
