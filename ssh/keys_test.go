package sshKey

import (
	"bytes"
	"encoding/pem"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateBBKeysRoundTrip(t *testing.T) {
	pair := GenerateBBKeys()
	block, rest := pem.Decode(pair.PrivateKey)
	if block == nil || block.Type != "OPENSSH PRIVATE KEY" || len(rest) != 0 {
		t.Fatal("private key is not an OpenSSH PEM block")
	}
	private, err := ssh.ParsePrivateKey(pair.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	public, _, _, rest, err := ssh.ParseAuthorizedKey(pair.PublicKey)
	if err != nil || len(rest) != 0 {
		t.Fatalf("invalid authorized public key: %v", err)
	}
	if public.Type() != ssh.KeyAlgoED25519 || !bytes.Equal(private.PublicKey().Marshal(), public.Marshal()) {
		t.Fatal("public key does not match the Ed25519 private key")
	}
}
