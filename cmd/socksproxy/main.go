package main

import (
	"log"

	sshproxy "github.com/gurupras/simple-socks-proxy"
)

func main() {
	hostKeyPath, err := sshproxy.DefaultHostKeyPath()
	if err != nil {
		log.Fatalf("Unable to determine default host key path: %v", err)
	}

	authorizedKeysPath, err := sshproxy.DefaultAuthorizedKeysPath()
	if err != nil {
		log.Fatalf("Failed to locate authorized_keys: %v", err)
	}

	err = sshproxy.StartServer(sshproxy.ServerConfig{
		ListenAddr:         "0.0.0.0:2222",
		HostPrivateKeyPath: hostKeyPath,
		AuthorizedKeysPath: authorizedKeysPath,
	})
	if err != nil {
		log.Fatal(err)
	}
}
