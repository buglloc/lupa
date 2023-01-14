package sshd

import "golang.org/x/crypto/ssh"

const (
	ExtensionPubFp = "pub-fp"
	ExtensionRole  = "role"
)

type HandlerFn func(conn *ssh.ServerConn, req interface{}) (interface{}, error)
