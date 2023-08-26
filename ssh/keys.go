package sshKey

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"

	"github.com/tez-capital/tezbake/util"

	"golang.org/x/crypto/ssh"
)

type Ed25519KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

func GenerateBBKeys() *Ed25519KeyPair {
	// Private Key generation
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	util.AssertE(err, "Failed to generate ed25519 key!")

	privateKeyEncoded := marshalED25519PrivateKey(privateKey)
	util.AssertE(err, "Failed to serialize ed25519 key!")

	return &Ed25519KeyPair{
		PublicKey:  marshalAuthorizedPublicKey(publicKey),
		PrivateKey: encodePrivateKeyToPEM(privateKeyEncoded),
	}
}

func marshalAuthorizedPublicKey(key ed25519.PublicKey) []byte {
	publicKey, err := ssh.NewPublicKey(key)
	util.AssertE(err, "Failed to preprocess ed25519 key!")
	publicKeySerialized := ssh.MarshalAuthorizedKey(publicKey)

	return publicKeySerialized
}

func encodePrivateKeyToPEM(bytes []byte) []byte {
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "OPENSSH PRIVATE KEY",
		Bytes: bytes,
	})
	return privatePEM
}

func IsValidSSHPublicKey(bytes []byte) bool {
	_, _, _, _, err := ssh.ParseAuthorizedKey(bytes)
	return err == nil
}

func IsValidSSHPrivateKey(bytes []byte) bool {
	_, err := ssh.ParsePrivateKey(bytes)
	return err == nil
}
