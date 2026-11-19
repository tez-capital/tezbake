package util

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"

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
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func unpad(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("invalid encrypted data padding")
	}
	unpadding := int(data[length-1])
	if unpadding < 1 || unpadding > aes.BlockSize || unpadding > length {
		return nil, errors.New("invalid encrypted data padding")
	}
	for _, value := range data[length-unpadding:] {
		if int(value) != unpadding {
			return nil, errors.New("invalid encrypted data padding")
		}
	}
	if !bytes.HasPrefix(data, []byte(AES_DATA_PREFIX)) {
		return nil, errors.New("failed to decrypt data")
	}
	if length-unpadding < len(AES_DATA_PREFIX) {
		return nil, errors.New("failed to decrypt data")
	}
	return data[len(AES_DATA_PREFIX):(length - unpadding)], nil
}

func DecryptAES(key, ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < 2*aes.BlockSize || len(ciphertext)%aes.BlockSize != 0 {
		return nil, errors.New("invalid ciphertext length")
	}
	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(block, iv)
	plaintext := make([]byte, len(ciphertext))
	mode.CryptBlocks(plaintext, ciphertext)
	return unpad(plaintext)
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
