package main

import (
	"log"
	"sync"

	simplesocksproxy "github.com/gurupras/simple-socks-proxy"
)

func main() {
	hostKeyPath, err := simplesocksproxy.DefaultHostKeyPath()
	if err != nil {
		log.Fatalf("Unable to determine default host key path: %v", err)
	}

	authorizedKeysPath, err := simplesocksproxy.DefaultAuthorizedKeysPath()
	if err != nil {
		log.Fatalf("Failed to locate authorized_keys: %v", err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	go func() {
		defer wg.Done()
		err = simplesocksproxy.StartServer(simplesocksproxy.ServerConfig{
			ListenAddr:         "0.0.0.0:2222",
			HostPrivateKeyPath: hostKeyPath,
			AuthorizedKeysPath: authorizedKeysPath,
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy := simplesocksproxy.DNSProxy{Addr: "0.0.0.0:5353"}
		if err := proxy.Start(); err != nil {
			log.Fatalf("Failed to start DNS proxy: %v\n", err)
		}
	}()

	wg.Wait()
}
