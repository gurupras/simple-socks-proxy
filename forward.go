package simplesocksproxy

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"

	"golang.org/x/crypto/ssh"
)

// Forwarder manages remote port forwards for a single SSH connection.
type Forwarder struct {
	mu        sync.Mutex
	listeners map[string]net.Listener
}

// NewForwarder creates a new Forwarder.
func NewForwarder() *Forwarder {
	return &Forwarder{
		listeners: make(map[string]net.Listener),
	}
}

// handleGlobalRequests handles incoming global requests for an SSH connection.
// It should be run in a separate goroutine.
// It handles "tcpip-forward" and "cancel-tcpip-forward" requests.
func handleGlobalRequests(conn *ssh.ServerConn, reqs <-chan *ssh.Request, forwards *Forwarder) {
	for req := range reqs {
		switch req.Type {
		case "tcpip-forward":
			go forwards.handleTCPIPForward(conn, req)
		case "cancel-tcpip-forward":
			go forwards.handleCancelTCPIPForward(req)
		default:
			// All other requests are not supported
			if req.WantReply {
				req.Reply(false, nil)
			}
		}
	}
}

// tcpipForwardPayload is the payload for a "tcpip-forward" request.
type tcpipForwardPayload struct {
	Addr string
	Port uint32
}

// handleTCPIPForward handles a "tcpip-forward" request.
func (f *Forwarder) handleTCPIPForward(conn *ssh.ServerConn, req *ssh.Request) {
	var payload tcpipForwardPayload
	if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
		log.Printf("Failed to unmarshal tcpip-forward payload: %v", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	// Note: payload.Addr can be empty, which means listen on all interfaces.
	addr := fmt.Sprintf("%s:%d", payload.Addr, payload.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("Failed to listen on %s: %v", addr, err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	listenAddr := ln.Addr().String()
	listenPort := uint32(ln.Addr().(*net.TCPAddr).Port)

	f.mu.Lock()
	f.listeners[listenAddr] = ln
	f.mu.Unlock()

	log.Printf("Listening on %s for remote forwarding to %s", listenAddr, conn.RemoteAddr())

	if req.WantReply {
		// Return the actual port we are listening on.
		// This is important if the client requested port 0.
		replyPayload := struct{ Port uint32 }{listenPort}
		req.Reply(true, ssh.Marshal(replyPayload))
	}

	// Start accepting connections in a new goroutine.
	go f.acceptConnections(conn, ln)
}

// handleCancelTCPIPForward handles a "cancel-tcpip-forward" request.
func (f *Forwarder) handleCancelTCPIPForward(req *ssh.Request) {
	var payload tcpipForwardPayload
	if err := ssh.Unmarshal(req.Payload, &payload); err != nil {
		log.Printf("Failed to unmarshal cancel-tcpip-forward payload: %v", err)
		if req.WantReply {
			req.Reply(false, nil)
		}
		return
	}

	addr := fmt.Sprintf("%s:%d", payload.Addr, payload.Port)

	f.mu.Lock()
	ln, ok := f.listeners[addr]
	if ok {
		delete(f.listeners, addr)
		ln.Close()
		log.Printf("Stopped listening on %s for remote forwarding", addr)
	}
	f.mu.Unlock()

	if req.WantReply {
		req.Reply(true, nil)
	}
}

// acceptConnections accepts connections on a listener and forwards them.
func (f *Forwarder) acceptConnections(conn *ssh.ServerConn, ln net.Listener) {
	listenAddr := ln.Addr().String()
	defer func() {
		f.mu.Lock()
		delete(f.listeners, listenAddr)
		f.mu.Unlock()
		ln.Close()
	}()

	for {
		tcpConn, err := ln.Accept()
		if err != nil {
			// This error is expected when the listener is closed.
			return
		}
		go f.forwardConnection(conn, tcpConn)
	}
}

// forwardConnection forwards a single TCP connection over a new SSH channel.
func (f *Forwarder) forwardConnection(conn *ssh.ServerConn, tcpConn net.Conn) {
	defer tcpConn.Close()

	localAddr := tcpConn.LocalAddr().(*net.TCPAddr)
	remoteAddr := tcpConn.RemoteAddr().(*net.TCPAddr)

	// Payload for a "forwarded-tcpip" channel.
	payload := ssh.Marshal(&struct {
		ConnectedAddr string
		ConnectedPort uint32
		OriginAddr    string
		OriginPort    uint32
	}{
		ConnectedAddr: localAddr.IP.String(),
		ConnectedPort: uint32(localAddr.Port),
		OriginAddr:    remoteAddr.IP.String(),
		OriginPort:    uint32(remoteAddr.Port),
	})

	channel, reqs, err := conn.OpenChannel("forwarded-tcpip", payload)
	if err != nil {
		log.Printf("Failed to open 'forwarded-tcpip' channel: %v", err)
		return
	}
	defer channel.Close()

	go ssh.DiscardRequests(reqs)

	// Bi-directional copy
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer channel.CloseWrite()
		io.Copy(channel, tcpConn)
	}()
	go func() {
		defer wg.Done()
		io.Copy(tcpConn, channel)
	}()

	wg.Wait()
}
