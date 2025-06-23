package simplesocksproxy

import (
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

type sshChannelConn struct {
	ssh.Channel
}

func (s *sshChannelConn) LocalAddr() net.Addr {
	return dummyAddr("ssh-local")
}

func (s *sshChannelConn) RemoteAddr() net.Addr {
	return dummyAddr("ssh-remote")
}

func (s *sshChannelConn) SetDeadline(t time.Time) error      { return nil }
func (s *sshChannelConn) SetReadDeadline(t time.Time) error  { return nil }
func (s *sshChannelConn) SetWriteDeadline(t time.Time) error { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return string(a) }
