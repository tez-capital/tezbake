package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	AES_DATA_PREFIX = "bake-buddy"
)

func PrepareAESKey(password string, salt []byte) []byte {
	const (
		time    = 1
		memory  = 64 * 1024
		threads = 4
	)

	return argon2.IDKey([]byte(password), salt, time, memory, threads, 32)
}

func pad(data []byte) []byte {
	data = append([]byte(AES_DATA_PREFIX), data...)
	blockSize := aes.BlockSize
	padding := blockSize - len(data)%blockSize
	if padding == 0 {
		padding = blockSize
	}
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}
func unpad(data []byte) ([]byte, error) {
	length := len(data)
	unpadding := int(data[length-1])
	if !strings.HasPrefix(string(data), AES_DATA_PREFIX) {
		return nil, errors.New("failed to decrypt data")
	}
	return data[len(AES_DATA_PREFIX):(length - unpadding)], nil
}

func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)
	return unpad(ciphertext)
}

func EncryptAES(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	data = pad(data)

	// The IV (Initialization Vector) needs to be unique, but not secure. It's common to include it at the beginning of the ciphertext.
	ciphertext := make([]byte, aes.BlockSize+len(data))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], data)
	return ciphertext, nil
}
