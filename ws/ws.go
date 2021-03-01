package ws

import (
	"bytes"
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/net/websocket"
)

type Client struct {
	l    sync.RWMutex
	c    *websocket.Config
	conn *websocket.Conn
	json *websocket.Codec
	last time.Time
}

func New(c *websocket.Config) *Client {
	return &Client{c: c, json: &websocket.Codec{jsonMarshal, jsonUnmarshal}}
}

func (c *Client) Connect() (*websocket.Conn, error) {
	c.l.RLock()
	conn := c.conn
	c.l.RUnlock()
	if conn != nil {
		return conn, nil
	}

	c.l.Lock()
	defer c.l.Unlock()
	if c.conn != nil {
		return c.conn, nil
	}

	for time.Since(c.last) < time.Second*2 {
		time.Sleep(time.Millisecond * 50)
	}
	c.last = time.Now()

	conn, err := websocket.DialConfig(c.c)
	if err != nil {
		return nil, err
	}
	c.conn = conn
	return conn, nil
}

func (c *Client) Disconnect() (err error) {
	c.l.Lock()
	if c.conn != nil {
		err = c.conn.Close()
		c.conn = nil
	}
	c.l.Unlock()
	return
}

func (c *Client) Read() (msg Message, err error) {
	var conn *websocket.Conn
	conn, err = c.Connect()
	if err != nil {
		return
	}

	err = c.json.Receive(conn, &msg)
	if err != nil {
		c.Disconnect()
	}
	return
}

func (c *Client) Send(v interface{}) error {
	conn, err := c.Connect()
	if err != nil {
		return err
	}

	return c.json.Send(conn, v)
}

func (c *Client) Subscribe(channel Channel) error {
	return c.Send(NewSubscribe(channel))
}

func (c *Client) Unsubscribe(channel Channel) error {
	return c.Send(NewUnsubscribe(channel))
}

func jsonMarshal(v interface{}) (msg []byte, payloadType byte, err error) {
	msg, err = json.Marshal(v)
	return msg, websocket.TextFrame, err
}

func jsonUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	d := json.NewDecoder(bytes.NewReader(msg))
	d.UseNumber()
	return d.Decode(v)
}
