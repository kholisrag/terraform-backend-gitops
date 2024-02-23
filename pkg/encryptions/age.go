package encryptions

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/kholisrag/terraform-backend-gitops/pkg/logger"
	"go.uber.org/zap"
)

func AgeEncrypt(publickey string, plaintext string, dst io.Writer) (err error) {
	recipient, err := age.ParseX25519Recipient(publickey)
	if err != nil {
		logger.Fatalf("failed to parse public key: %v", err)
	}

	w, err := age.Encrypt(dst, recipient)
	if err != nil {
		logger.Fatalf("failed to create encrypted file: %v", err)
	}
	if _, err := io.WriteString(w, plaintext); err != nil {
		logger.Fatalf("failed to write to encrypted file: %v", err)
	}
	if err := w.Close(); err != nil {
		logger.Fatalf("failed to close encrypted file: %v", err)
	}
	return err
}

func AgeDecrypt(privateKeyPath string, filePath string) (result map[string]interface{}, err error) {
	logger.Debugf("privateKeyPath: %s", privateKeyPath)
	privateKeyFile, err := os.Open(privateKeyPath)
	if err != nil {
		logger.Fatalf("failed to read private key: %v", err)
	}
	defer privateKeyFile.Close()

	var privateKey string
	scanner := bufio.NewScanner(privateKeyFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "AGE-SECRET-KEY-") {
			logger.Debug("found private key")
			privateKey = line // pragma: allowlist secret
			break
		}
	}

	if privateKey != "" {
		logger.Debugf("age private key found in : %s", privateKeyPath)
	} else {
		logger.Fatalf("age private key not found")
		return result, err
	}

	identity, err := age.ParseX25519Identity(privateKey)
	if err != nil {
		logger.Fatalf("failed to parse private key: %v", err)
		return result, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		// logger.Errorf("failed to open file: %v", err)
		logger.Error("failed to open file", zap.Error(err))
		return result, err
	}
	defer file.Close()

	decryptedIOReader, err := age.Decrypt(file, identity)
	if err != nil {
		logger.Fatalf("failed to open encrypted file: %v", err)
		return result, err
	}

	dataByte, err := io.ReadAll(decryptedIOReader)
	if err != nil {
		return nil, err
	}

	dataStringNormalized := strings.ReplaceAll(string(dataByte), "\n", "")
	err = json.Unmarshal([]byte(dataStringNormalized), &result)
	if err != nil {
		return nil, err
	}
	return result, err
}
