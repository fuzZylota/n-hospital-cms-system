package test

import (
	"fmt"
	"lib"
	"os"
	"testing"
)

func TestEncryptAndDecrypt(t *testing.T) {
	err := os.Setenv("ENCRYPTION_KEY", "aksjkdhgkaldjgbh")

	if err != nil {
		t.Fatalf("Error setting encryption key: %v", err)
	}

	plaintext := "Admin123"

	encrypted, err := lib.Encrypt([]byte(plaintext))

	if err != nil {
		t.Fatalf("Error encrypting: %v", err)
	}

	fmt.Printf("Encrypted: %s\n", encrypted)

	decrypted, err := lib.Decrypt(encrypted)

	if err != nil {
		t.Fatalf("Error decrypting: %v", err)
	}

	fmt.Printf("Decrypted: %s\n", string(decrypted))
}
