package simplesocksproxy

import (
	"fmt"
	"io"
	"log"
	"net"

	"github.com/armon/go-socks5"
	"golang.org/x/crypto/ssh"
)

type ServerConfig struct {
	ListenAddr         string
	HostPrivateKeyPath string
	AuthorizedKeysPath string
}

func StartServer(cfg ServerConfig) error {
	pubKeyCallback, err := PublicKeyAuthCallback(cfg.AuthorizedKeysPath)
	if err != nil {
		return err
	}

	hostKeySigner, err := LoadOrCreateHostKey(cfg.HostPrivateKeyPath)
	if err != nil {
		return err
	}

	sshCfg := &ssh.ServerConfig{
		PublicKeyCallback: pubKeyCallback,
	}
	sshCfg.AddHostKey(hostKeySigner)

	listener, err := net.Listen("tcp", cfg.ListenAddr)
	if err != nil {
		return err
	}
	log.Printf("Listening on %s...", cfg.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Failed to accept incoming connection:", err)
			continue
		}

		go handleConn(conn, sshCfg)
	}
}

func handleConn(netConn net.Conn, sshCfg *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, sshCfg)
	if err != nil {
		log.Println("SSH handshake failed:", err)
		return
	}

	log.Printf("New SSH connection from %s", sshConn.RemoteAddr())

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "direct-tcpip" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		var payload struct {
			DestAddr string
			DestPort uint32
			OrigAddr string
			OrigPort uint32
		}
		ssh.Unmarshal(newChannel.ExtraData(), &payload)

		dest := fmt.Sprintf("%s:%d", payload.DestAddr, payload.DestPort)
		log.Printf("Forwarding to %s", dest)

		upstream, err := net.Dial("tcp", dest)
		if err != nil {
			newChannel.Reject(ssh.ConnectionFailed, err.Error())
			continue
		}

		channel, _, err := newChannel.Accept()
		if err != nil {
			log.Println("Channel accept error:", err)
			upstream.Close()
			continue
		}

		// Bi-directional copy
		go func() {
			defer channel.Close()
			defer upstream.Close()
			io.Copy(channel, upstream)
		}()
		go func() {
			defer channel.Close()
			defer upstream.Close()
			io.Copy(upstream, channel)
		}()
	}
}

func handleSOCKS(channel ssh.Channel, requests <-chan *ssh.Request) {
	go func() {
		for req := range requests {
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}()

	conf := &socks5.Config{}
	server, err := socks5.New(conf)
	if err != nil {
		log.Println("SOCKS5 server setup failed:", err)
		channel.Close()
		return
	}

	wrappedConn := &sshChannelConn{Channel: channel}
	if err := server.ServeConn(wrappedConn); err != nil && err != io.EOF {
		log.Println("SOCKS5 error:", err)
	}

	_ = channel.Close()
}
