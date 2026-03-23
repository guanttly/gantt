package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

const AESKey = "jusha1996"

// 生成128位的MD5哈希
func GenMD5Hash(key string) []byte {
	hash := md5.Sum([]byte(key))
	return hash[:]
}

// 使用AES密钥解密字节数组
func AESDecryptBytes(content []byte, key string) ([]byte, error) {
	// 将密钥转换为字节数组
	keyBytes := GenMD5Hash(key)

	// 创建 AES 加密块
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	// 创建 GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 检查加密内容长度
	nonceSize := aesGCM.NonceSize()
	if len(content) < nonceSize {
		return nil, errors.New("cipher text too short")
	}

	// 分离 Nonce 和加密内容
	nonce, cipherText := content[:nonceSize], content[nonceSize:]

	// 解密内容
	plainText, err := aesGCM.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, err
	}

	return plainText, nil
}

// 使用AES密钥解密字符串
func AESDecrypt(content []byte, key string) (string, error) {
	// 使用AES密钥解密文件内容
	decryptBytes, err := AESDecryptBytes(content, key)
	if err != nil {
		return "", err
	}
	return string(decryptBytes), nil
}

// 解密服务间加密特性
func AESDecryptServerFeature(encryptStr string) (string, error) {
	// base64解码
	encryptBytes, err := Base64Decode(encryptStr)
	if err != nil {
		return "", errors.New("base64 decode error")
	}
	// AES解密
	decryptStr, err := AESDecrypt(encryptBytes, AESKey)
	if err != nil {
		return "", errors.New("decrypt error")
	}
	return decryptStr, nil
}

// 使用AES密钥加密字节数组
func AESEncryptBytes(content []byte, key string) ([]byte, error) {
	// 将密钥转换为字节数组
	keyBytes := GenMD5Hash(key)

	// 创建 AES 加密块
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	// 创建 GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	// 生成随机的 Nonce
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// 加密内容
	cipherText := aesGCM.Seal(nonce, nonce, content, nil)
	return cipherText, nil
}

// 加密服务间加密特性
func AESEncryptServerFeature(content string) (string, error) {
	// AES加密
	encryptBinary, err := AESEncryptBytes([]byte(content), AESKey)
	if err != nil {
		return "", errors.New("encrypt error")
	}
	return Base64Encode(encryptBinary), nil
}

// base64编码
func Base64Encode(src []byte) string {
	return base64.StdEncoding.EncodeToString(src)
}

// base64解码
func Base64Decode(src string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(src)
}
