package lupad

import (
	"errors"
	"fmt"

	"github.com/gofrs/uuid"
	"golang.org/x/crypto/ssh"

	"github.com/buglloc/lupa/internal/mdb"
	"github.com/buglloc/lupa/internal/sshd"
	"github.com/buglloc/lupa/pkg/lupa"
)

type SSHToMDB struct {
	mdb *mdb.MachineDB
}

func BindHandlers(mdb *mdb.MachineDB, sshSrv *sshd.Server) *SSHToMDB {
	out := &SSHToMDB{
		mdb: mdb,
	}

	sshSrv.AddHandler("get", out.Get)
	sshSrv.AddHandler("put", out.Put)
	return out
}

func (s *SSHToMDB) Get(conn *ssh.ServerConn, msg interface{}) (interface{}, error) {
	machineFP, err := sshConToMachineFP(conn)
	if err != nil {
		return nil, err
	}

	req, ok := msg.(*lupa.GetReqMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected request type: %T", req)
	}

	out, err := s.mdb.Get(machineFP, req.KeyID)
	if err != nil {
		return nil, fmt.Errorf("unable to get data: %w", err)
	}

	return &lupa.GetRspMsg{
		Data: out,
	}, nil
}

func (s *SSHToMDB) Put(conn *ssh.ServerConn, msg interface{}) (interface{}, error) {
	machineFP, err := sshConToMachineFP(conn)
	if err != nil {
		return nil, err
	}

	req, ok := msg.(*lupa.PutReqMsg)
	if !ok {
		return nil, fmt.Errorf("unexpected request type: %T", req)
	}

	keyUUID, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("unable to generate key id: %w", err)
	}

	keyID := keyUUID.String()
	if err := s.mdb.Put(machineFP, keyID, req.Data); err != nil {
		return nil, fmt.Errorf("unable to store data: %w", err)
	}

	return &lupa.PutRspMsg{
		KeyID: keyID,
	}, nil
}

func sshConToMachineFP(conn *ssh.ServerConn) (string, error) {
	return sshConExtension(conn, sshd.ExtensionPubFp)
}

func sshConExtension(conn *ssh.ServerConn, key string) (string, error) {
	if conn.Permissions == nil {
		return "", errors.New("no SSH permissions was set")
	}

	out, ok := conn.Permissions.Extensions[key]
	if !ok {
		return "", fmt.Errorf("no SSH extension %q was set", key)
	}

	return out, nil
}
