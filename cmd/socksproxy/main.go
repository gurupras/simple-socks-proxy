package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"sync"

	simplesocksproxy "github.com/gurupras/simple-socks-proxy"
)

var pacFilePaths []string

func main() {
	defaultHostKeyPath, err := simplesocksproxy.DefaultHostKeyPath()
	if err != nil {
		log.Fatalf("Unable to determine default host key path: %v", err)
	}

	defaultAuthorizedKeysPath, err := simplesocksproxy.DefaultAuthorizedKeysPath()
	if err != nil {
		log.Fatalf("Failed to locate authorized_keys: %v", err)
	}

	listenAddr := flag.String("listen-addr", "0.0.0.0:2222", "SSH SOCKS5 proxy listen address")
	hostKey := flag.String("host-key", defaultHostKeyPath, "Path to SSH host private key")
	authorizedKeys := flag.String("authorized-keys", defaultAuthorizedKeysPath, "Path to authorized_keys file")
	dnsAddr := flag.String("dns-addr", "0.0.0.0:53530", "DNS proxy listen address")

	pacAddr := flag.String("pac-addr", "0.0.0.0:2223", "HTTP port to serve PAC files")
	pacBasePath := flag.String("pac-path", "/pac", "Base URL path for PAC files")

	flag.Func("pac", "PAC file(s) to serve via HTTP (can be specified multiple times)", func(p string) error {
		pacFilePaths = append(pacFilePaths, p)
		return nil
	})

	flag.Parse()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := simplesocksproxy.StartServer(simplesocksproxy.ServerConfig{
			ListenAddr:         *listenAddr,
			HostPrivateKeyPath: *hostKey,
			AuthorizedKeysPath: *authorizedKeys,
		})
		if err != nil {
			log.Fatal(err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		proxy := simplesocksproxy.DNSProxy{Addr: *dnsAddr}
		if err := proxy.Start(); err != nil {
			log.Fatalf("Failed to start DNS proxy: %v\n", err)
		}
	}()

	if len(pacFilePaths) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			base := *pacBasePath
			if !strings.HasPrefix(base, "/") {
				base = "/" + base
			}
			base = strings.TrimRight(base, "/")

			mux := http.NewServeMux()
			for _, path := range pacFilePaths {
				filename := filepath.Base(path)
				route := base + "/" + filename
				mux.HandleFunc(route, func(path string) http.HandlerFunc {
					return func(w http.ResponseWriter, r *http.Request) {
						http.ServeFile(w, r, path)
					}
				}(path))
				log.Printf("Serving PAC file %s at http://%v%s\n", path, *pacAddr, route)
			}

			addr := fmt.Sprintf("%v", *pacAddr)
			log.Fatal(http.ListenAndServe(addr, mux))
		}()
	}

	wg.Wait()
}
