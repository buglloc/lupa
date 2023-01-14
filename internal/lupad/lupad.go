package lupad

import (
	"context"
	"fmt"

	"golang.org/x/crypto/ssh"

	"github.com/buglloc/lupa/internal/config"
	"github.com/buglloc/lupa/internal/mdb"
	"github.com/buglloc/lupa/internal/sshd"
)

type Server struct {
	sshd    *sshd.Server
	handler *SSHToMDB
	mdb     *mdb.MachineDB
	cfg     *config.Config
}

func NewServer(cfg *config.Config) (*Server, error) {
	srv := &Server{
		cfg: cfg,
	}

	var err error
	srv.sshd, err = sshd.NewServer(&sshd.Config{
		SSH:          cfg.SSH,
		CheckUserKey: srv.publicKeyCallback,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to create SSHD server: %w", err)
	}

	srv.mdb, err = mdb.NewMachineDB(cfg.DB.StorePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create DB: %w", err)
	}

	srv.handler = BindHandlers(srv.mdb, srv.sshd)
	return srv, nil
}

func (s *Server) ListenAndServe() error {
	return s.sshd.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.sshd.Shutdown(ctx)
}

func (s *Server) publicKeyCallback(user string, pubKey ssh.PublicKey) (string, error) {
	targetFp := ssh.FingerprintSHA256(pubKey)
	userInfo, ok := s.cfg.Users[user]
	if ok {
		for _, key := range userInfo.SHA256Keys {
			if targetFp == key {
				return userInfo.Role, nil
			}
		}

		return RoleNone, fmt.Errorf("unknown key %s", targetFp)
	}

	if s.cfg.AllowRegistration || s.mdb.IsMachineExists(targetFp) {
		return RoleUser, nil
	}
	return RoleNone, fmt.Errorf("user %q was not found", user)
}
