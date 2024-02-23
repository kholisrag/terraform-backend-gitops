package encryptions

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"testing"

	"filippo.io/age"
)

func TestAgeEncryptAndDecrypt(t *testing.T) {
	// Generate a new age key pair
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("Failed to generate age key pair: %v", err)
	}

	publicKey := identity.Recipient().String()
	privateKey := identity.String()

	// Write the private key to a file
	tmpPrivateKey, err := os.CreateTemp("", "agetestprivatekey")
	if err != nil {
		log.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpPrivateKey.Name())

	err = os.WriteFile(tmpPrivateKey.Name(), []byte(privateKey), 0600)
	if err != nil {
		log.Fatalf("failed to write private key to file: %v", err)
	}

	plaintext := `{"data": "your-text-for-unit-test-here"}`

	// Test AgeEncrypt
	var encrypted bytes.Buffer
	err = AgeEncrypt(publicKey, plaintext, &encrypted)
	if err != nil {
		t.Errorf("AgeEncrypt returned an error: %v", err)
	}

	// Write the encrypted string to a file
	tmpEncryptedData, err := os.CreateTemp("", "agetestencrypteddata")
	if err != nil {
		log.Fatalf("failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpEncryptedData.Name())

	err = os.WriteFile(tmpEncryptedData.Name(), []byte(encrypted.Bytes()), 0600)
	if err != nil {
		log.Fatalf("failed to write encrypted string to file: %v", err)
	}

	// Test AgeDecrypt
	result, err := AgeDecrypt(tmpPrivateKey.Name(), tmpEncryptedData.Name())
	if err != nil {
		t.Errorf("AgeDecrypt returned an error: %v", err)
	}
	resultMarshal, err := json.Marshal(result)
	if err != nil {
		t.Errorf("failed to marshal decrypted data: %v", err)
	}

	// Check if the decrypted text is the same as the original plaintext
	var decrypted map[string]interface{}
	err = json.Unmarshal(resultMarshal, &decrypted)
	if err != nil {
		t.Errorf("failed to unmarshal decrypted data: %v", err)
	}
	if decrypted["data"] != "your-text-for-unit-test-here" {
		t.Errorf("decrypted data is not the same as the original plaintext")
	} else {
		t.Logf("decrypted data is the same as the original plaintext: %v", decrypted["data"])
	}
}
