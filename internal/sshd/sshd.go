package sshd

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"

	"github.com/buglloc/lupa/internal/config"
	"github.com/buglloc/lupa/pkg/lupa"
)

type Config struct {
	config.SSH
	CheckUserKey func(user string, pubKey ssh.PublicKey) (string, error)
}

type Server struct {
	addr       string
	listener   net.Listener
	sshConf    ssh.ServerConfig
	handlers   map[string]HandlerFn
	checkKeyFn func(user string, pubKey ssh.PublicKey) (string, error)
	closed     chan struct{}
	ctx        context.Context
	shutdownFn context.CancelFunc
}

func NewServer(cfg *Config) (*Server, error) {
	srv := &Server{
		addr:       cfg.Addr,
		handlers:   make(map[string]HandlerFn),
		checkKeyFn: cfg.CheckUserKey,
		closed:     make(chan struct{}),
	}

	srv.sshConf = ssh.ServerConfig{
		MaxAuthTries:      -1,
		ServerVersion:     "SSH-2.0-Lupad",
		PublicKeyCallback: srv.publicKeyCallback,
		AuthLogCallback: func(sshConn ssh.ConnMetadata, method string, err error) {
			if method == "none" {
				// huh
				return
			}

			if err != nil {
				log.Warn().
					Str("remote_addr", sshConn.RemoteAddr().String()).
					Str("session_id", newSessID(sshConn.SessionID())).
					Str("auth_method", method).
					Str("user", sshConn.User()).
					Err(err).Msg("auth failed")
				return
			}

			log.Warn().
				Str("remote_addr", sshConn.RemoteAddr().String()).
				Str("session_id", newSessID(sshConn.SessionID())).
				Str("auth_method", method).
				Str("user", sshConn.User()).
				Msg("user authenticated")
		},
	}

	haveKey := false
	for _, keyPath := range cfg.HostKeys {
		rawKey, err := os.ReadFile(keyPath)
		if err != nil {
			log.Info().Str("key_path", keyPath).Err(err).Msg("ignore host key")
			continue
		}

		key, err := ssh.ParsePrivateKey(rawKey)
		if err != nil {
			log.Warn().Str("key_path", keyPath).Err(err).Msg("skip malformed host key")
			continue
		}

		srv.sshConf.AddHostKey(key)
		haveKey = true
	}

	if !haveKey {
		return nil, errors.New("no host keys found")
	}

	srv.ctx, srv.shutdownFn = context.WithCancel(context.Background())
	return srv, nil
}

func (s *Server) ListenAndServe() error {
	defer close(s.closed)

	listener, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	log.Info().
		Str("addr", s.addr).
		Msg("listening")

	for {
		tcpConn, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}

			log.Warn().Err(err).Msg("unable to accept incoming connection")
			continue
		}

		go s.acceptConnection(tcpConn)
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	_ = s.listener.Close()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-s.closed:
		return nil
	}
}

func (s *Server) AddHandler(name string, fn HandlerFn) {
	s.handlers[name] = fn
}

func (s *Server) publicKeyCallback(conn ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
	if s.checkKeyFn == nil {
		return nil, errors.New("CheckUserKey handler is not configured")
	}

	role, err := s.checkKeyFn(conn.User(), pubKey)
	if err != nil {
		return nil, err
	}

	return &ssh.Permissions{
		Extensions: map[string]string{
			ExtensionPubFp: ssh.FingerprintSHA256(pubKey),
			ExtensionRole:  role,
		},
	}, nil
}

// Accept a single connection - run in a go routine as the ssh authentication can block
func (s *Server) acceptConnection(conn net.Conn) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, &s.sshConf)
	if err != nil {
		switch {
		case errors.Is(err, io.EOF):
		case strings.Contains(strings.ToLower(err.Error()), "connection reset by peer"):
		default:
			log.Error().Err(err).Msg("SSH handshake failed")
		}
		return
	}

	// Discard all global out-of-band Requests
	go ssh.DiscardRequests(reqs)
	// Accept all channels
	go s.handleChannels(sshConn, chans)
}

func (s *Server) handleChannels(sshConn *ssh.ServerConn, chans <-chan ssh.NewChannel) {
	logger := log.With().
		Str("remote_addr", sshConn.RemoteAddr().String()).
		Str("session_id", newSessID(sshConn.SessionID())).
		Str("user", sshConn.User()).
		Logger()

	for newChannel := range chans {
		if newChannel.ChannelType() != lupa.ChannelType {
			_ = newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		func() {
			channel, reqs, err := newChannel.Accept()
			if err != nil {
				logger.Warn().Err(err).Msg("unable to accept lupa channel")
				return
			}
			defer func() { _ = channel.Close() }()

			// Discard all global out-of-band Requests
			go ssh.DiscardRequests(reqs)

			lupaChan := lupa.NewChannel(channel)
			for {
				err := lupaChan.ProcessRequest(func(typ string, msg interface{}) (interface{}, error) {
					rsp, err := s.handleReq(sshConn, typ, msg)
					if err != nil {
						logger.Warn().Str("req_type", typ).Err(err).Msg("unable to process request")
					} else {
						logger.Info().Str("req_type", typ).Msg("request processed")
					}
					return rsp, err
				})

				if err != nil {
					logger.Warn().Err(err).Msg("unable to process channel requests")
					break
				}
			}
		}()
	}
}

func (s *Server) handleReq(conn *ssh.ServerConn, typ string, msg interface{}) (interface{}, error) {
	handler, ok := s.handlers[typ]
	if !ok {
		return nil, fmt.Errorf("unsupported request: %s", typ)
	}

	return handler(conn, msg)
}

func newSessID(sshSessionID []byte) string {
	return hex.EncodeToString(sshSessionID)
}
