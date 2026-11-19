package util

import (
	"bytes"
	"testing"
)

func TestCredentialDecryptionRejectsCorruption(t *testing.T) {
	key := bytes.Repeat([]byte{1}, 32)
	for _, length := range []int{0, 1, 15, 16, 17, 31, 33} {
		if _, err := DecryptAES(key, make([]byte, length)); err == nil {
			t.Fatalf("accepted ciphertext length %d", length)
		}
	}
	for _, data := range [][]byte{nil, []byte("bake-buddy\x00"), []byte("bake-buddy\x02\x03"), []byte("bake-buddy\xff")} {
		if _, err := unpad(data); err == nil {
			t.Fatal("accepted invalid padding")
		}
	}
	want := []byte(`{"password":"synthetic test credential"}`)
	encrypted, err := EncryptAES(key, want)
	if err != nil {
		t.Fatal(err)
	}
	original := bytes.Clone(encrypted)
	_, _ = DecryptAES(bytes.Repeat([]byte{2}, 32), encrypted)
	got, err := DecryptAES(key, encrypted)
	if err != nil || !bytes.Equal(got, want) {
		t.Fatalf("round trip: %q, %v", got, err)
	}
	if !bytes.Equal(encrypted, original) {
		t.Fatal("credential decryption changed the ciphertext used by password retries")
	}
}
