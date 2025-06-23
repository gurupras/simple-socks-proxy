package simplesocksproxy

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/fs"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func DefaultHostKeyPath() (string, error) {
	var baseDir string

	// fallback to user's home config dir
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	baseDir = filepath.Join(home, ".config", "simplesocksproxy")

	// Ensure the directory exists
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return "", err
	}

	return filepath.Join(baseDir, "ssh_host_rsa_key"), nil
}

func LoadOrCreateHostKey(path string) (ssh.Signer, error) {
	keyBytes, err := os.ReadFile(path)
	if err == nil {
		return ssh.ParsePrivateKey(keyBytes)
	}

	// Generate new RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	if err := os.WriteFile(path, keyPEM, fs.FileMode(0600)); err != nil {
		return nil, err
	}

	return ssh.NewSignerFromKey(privateKey)
}
