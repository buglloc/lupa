package lupa

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/ssh"
)

type CallMsg struct {
	Type    string
	Payload []byte `ssh:"rest"`
}

const failureMsgType = 100

type FailureMsg struct {
	Msg string `sshtype:"100"`
}

const successMsgType = 101

type SuccessMsg struct{}

const putReqMsgType = 110

type PutReqMsg struct {
	Data []byte `sshtype:"110"`
}

const putRspMsgType = 111

type PutRspMsg struct {
	KeyID string `sshtype:"111"`
}

const getReqMsgType = 112

type GetReqMsg struct {
	KeyID string `sshtype:"112"`
}

const getRspMsgType = 113

type GetRspMsg struct {
	Data []byte `sshtype:"113"`
}

func UnmarshalMsg(packet []byte) (interface{}, error) {
	if len(packet) < 1 {
		return nil, errors.New("empty packet")
	}

	var msg interface{}
	switch packet[0] {
	case successMsgType:
		return new(SuccessMsg), nil
	case failureMsgType:
		msg = new(FailureMsg)
	case putReqMsgType:
		msg = new(PutReqMsg)
	case putRspMsgType:
		msg = new(PutRspMsg)
	case getReqMsgType:
		msg = new(GetReqMsg)
	case getRspMsgType:
		msg = new(GetRspMsg)
	default:
		return nil, fmt.Errorf("agent: unknown type tag %d", packet[0])
	}
	if err := ssh.Unmarshal(packet, msg); err != nil {
		return nil, err
	}
	return msg, nil
}
