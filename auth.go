package simplesocksproxy

import (
	"errors"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

type PublicKeyCallback func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error)

func PublicKeyAuthCallback(authorizedKeysPath string) (PublicKeyCallback, error) {
	content, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		return nil, err
	}

	authorizedKeysMap := map[string]bool{}
	for len(content) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(content)
		if err != nil {
			return nil, errors.New("failed to parse authorized_keys")
		}
		authorizedKeysMap[string(pubKey.Marshal())] = true
		content = rest
	}

	return func(conn ssh.ConnMetadata, key ssh.PublicKey) (*ssh.Permissions, error) {
		if authorizedKeysMap[string(key.Marshal())] {
			return nil, nil
		}
		return nil, errors.New("unauthorized public key")
	}, nil
}

// Cross-platform expansion of "~/.ssh/authorized_keys"
func DefaultAuthorizedKeysPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".ssh", "authorized_keys"), nil
}
