package lupa

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type Client struct {
	ch *Channel
}

func NewClient(sshc *ssh.Client) (*Client, error) {
	ch, reqs, err := sshc.OpenChannel(ChannelType, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to request lupa channel: %w", err)
	}

	// Discard all global out-of-band Requests
	go ssh.DiscardRequests(reqs)

	return &Client{
		ch: NewChannel(ch),
	}, nil
}

func (c *Client) Get(keyID string) ([]byte, error) {
	rsp, err := c.ch.Call("get", &GetReqMsg{
		KeyID: keyID,
	})
	if err != nil {
		return nil, err
	}

	getRsp, ok := rsp.(*GetRspMsg)
	if !ok {
		return nil, fmt.Errorf("unexptected response type %T", rsp)
	}

	return getRsp.Data, nil
}

func (c *Client) Put(data []byte) (string, error) {
	rsp, err := c.ch.Call("put", &PutReqMsg{
		Data: data,
	})
	if err != nil {
		return "", err
	}

	putRsp, ok := rsp.(*PutRspMsg)
	if !ok {
		return "", fmt.Errorf("unexptected response type %T", rsp)
	}

	return putRsp.KeyID, nil
}

func (c *Client) Close() error {
	return c.ch.Close()
}
