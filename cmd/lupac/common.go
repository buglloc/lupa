package main

import (
	"fmt"
	"net"
	"os"
	"time"

	"golang.org/x/crypto/ssh"

	"github.com/buglloc/lupa/pkg/lupa"
)

func dial() (*lupa.Client, func(), error) {
	privBytes, err := os.ReadFile(rootArgs.PrivateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to read private key file: %w", err)
	}

	signer, err := ssh.ParsePrivateKey(privBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key signer: %w", err)
	}

	config := &ssh.ClientConfig{
		User:    ssh.FingerprintSHA256(signer.PublicKey()),
		Timeout: 5 * time.Second,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: func(_ string, _ net.Addr, key ssh.PublicKey) error {
			if rootArgs.RemoteFingerprint == "" {
				return nil
			}

			actualFp := ssh.FingerprintSHA256(key)
			if rootArgs.RemoteFingerprint == actualFp {
				return nil
			}

			return fmt.Errorf("remote host key mismatch: %q (expected) != %q (actual)", rootArgs.RemoteFingerprint, actualFp)
		},
	}

	conn, err := ssh.Dial("tcp", rootArgs.RemoteAddr, config)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to connect: %w", err)
	}

	lupac, err := lupa.NewClient(conn)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create lupa client: %w", err)
	}

	closeFn := func() {
		_ = lupac.Close()
		_ = conn.Close()
	}
	return lupac, closeFn, nil
}
