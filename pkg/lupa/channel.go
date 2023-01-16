package lupa

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	"golang.org/x/crypto/ssh"
)

const ChannelType = "lupa@buglloc.com"
const maxChannelBytes = 64 << 10

type Channel struct {
	closed bool
	mu     sync.Mutex
	conn   io.ReadWriteCloser
}

func NewChannel(conn io.ReadWriteCloser) *Channel {
	return &Channel{
		conn: conn,
	}
}

func (c *Channel) ProcessRequest(fn func(string, interface{}) (interface{}, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return errors.New("channel closed")
	}

	err := func() error {
		var length [4]byte
		if _, err := io.ReadFull(c.conn, length[:]); err != nil {
			return err
		}
		l := binary.BigEndian.Uint32(length[:])
		if l == 0 {
			return fmt.Errorf("request size is 0")
		}

		if l > maxChannelBytes {
			// We also cap requests.
			return fmt.Errorf("request too large: %d", l)
		}

		buf := make([]byte, l)
		if _, err := io.ReadFull(c.conn, buf); err != nil {
			return err
		}

		var callMsg CallMsg
		if err := ssh.Unmarshal(buf, &callMsg); err != nil {
			return fmt.Errorf("unexpected request: %w", err)
		}

		req, err := UnmarshalMsg(callMsg.Payload)
		if err != nil {
			return fmt.Errorf("unexpected request: %w", err)
		}

		rsp, err := fn(callMsg.Type, req)
		if err != nil {
			rsp = &FailureMsg{
				Msg: err.Error(),
			}
		}

		rspData := ssh.Marshal(rsp)
		if len(rspData) > maxChannelBytes {
			return fmt.Errorf("reply too large: %d bytes", len(rspData))
		}

		binary.BigEndian.PutUint32(length[:], uint32(len(rspData)))
		if _, err := c.conn.Write(length[:]); err != nil {
			return err
		}

		if _, err := c.conn.Write(rspData); err != nil {
			return err
		}

		return nil
	}()

	if err != nil {
		_ = c.closeLocked()
	}
	return err
}

func (c *Channel) Call(typ string, req interface{}) (reply interface{}, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	callMsg := &CallMsg{
		Type:    typ,
		Payload: ssh.Marshal(req),
	}
	buf, err := c.callRawLocked(ssh.Marshal(callMsg))
	if err != nil {
		_ = c.closeLocked()
		return nil, err
	}

	reply, err = UnmarshalMsg(buf)
	if err != nil {
		return nil, fmt.Errorf("unexpected response: %w", err)
	}

	if fail, ok := reply.(*FailureMsg); ok {
		return nil, fmt.Errorf("remote error: %s", fail.Msg)
	}

	return reply, nil
}

func (c *Channel) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.closeLocked()
}

func (c *Channel) closeLocked() error {
	c.closed = true
	return c.conn.Close()
}

func (c *Channel) callRawLocked(req []byte) ([]byte, error) {
	if c.closed {
		return nil, errors.New("channel closed")
	}

	msg := make([]byte, 4+len(req))
	binary.BigEndian.PutUint32(msg, uint32(len(req)))
	copy(msg[4:], req)
	if _, err := c.conn.Write(msg); err != nil {
		return nil, fmt.Errorf("write: %w", err)
	}

	var respSizeBuf [4]byte
	if _, err := io.ReadFull(c.conn, respSizeBuf[:]); err != nil {
		return nil, fmt.Errorf("read response size: %w", err)
	}
	respSize := binary.BigEndian.Uint32(respSizeBuf[:])
	if respSize > maxChannelBytes {
		return nil, errors.New("response too large")
	}

	buf := make([]byte, respSize)
	if _, err := io.ReadFull(c.conn, buf); err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	return buf, nil
}
