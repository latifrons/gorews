package gorews

import (
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"
)

const ChanBufferSize = 64

type GorewsClient struct {
	Incoming         chan []byte
	Outgoing         chan []byte
	sending          []byte
	connected        bool
	quit             chan bool
	quited           bool
	Url              *url.URL
	ReqHeader        http.Header
	Dialer           *websocket.Dialer
	connection       *websocket.Conn
	handshakeTimeout time.Duration
	writeTimeout     time.Duration
	readTimeout      time.Duration
}

func NewGorewsClient() *GorewsClient {
	return &GorewsClient{
		Incoming: make(chan []byte, ChanBufferSize),
		Outgoing: make(chan []byte, ChanBufferSize),
		quit:     make(chan bool),
	}
}

func (c *GorewsClient) Start(urlStr string, reqHeader http.Header,
	handshakeTimeout time.Duration, writeTimeout time.Duration, readTimeout time.Duration) error {
	url, err := url.Parse(urlStr)

	if err != nil {
		return err
	}
	c.Url = url
	c.ReqHeader = reqHeader
	c.handshakeTimeout = handshakeTimeout
	c.writeTimeout = writeTimeout
	c.readTimeout = readTimeout
	c.Dialer = &websocket.Dialer{
		HandshakeTimeout: handshakeTimeout,
		//Proxy:            rc.Proxy,
	}

	c.newConnection()
	go c.pingPong()
	// wait until connection is built
	go c.write()
	go c.read()
	for {
		time.Sleep(time.Second)
		c.Outgoing <- []byte(time.Now().String())
	}
	return nil
}

func (c *GorewsClient) newConnection() {
	// build connection until success
	c.connected = false
	if c.connection != nil {
		err := c.connection.Close()
		if err != nil {
			logrus.WithError(err).Debug("error on closing connection")
		}
	}
	for {
		logrus.Info("building new connection")
		wsConn, _, err := c.Dialer.Dial(c.Url.String(), c.ReqHeader)
		if err != nil {
			logrus.WithError(err).Warn("error on building new connection")
			time.Sleep(time.Second)
			continue
		}
		c.connection = wsConn
		c.connected = true
		return
	}
}

func (c *GorewsClient) Stop() {
	c.quited = true
	close(c.quit)
}

func (c *GorewsClient) IsConnected() bool {
	return c.connected
}

func (c *GorewsClient) write() {
	for {
		// consume a message from the channel
		select {
		case msg := <-c.Outgoing:
			// send until success
			c.sending = msg
			for {
				if !c.connected {
					time.Sleep(time.Second)
					continue
				}
				// set timeout
				_ = c.connection.SetWriteDeadline(time.Now().Add(c.writeTimeout))
				err := c.connection.WriteMessage(websocket.TextMessage, msg)
				if err != nil {
					// connection may be broken. reset
					logrus.WithError(err).Warn("write error")
					c.connected = false
					time.Sleep(time.Second)
				} else {
					break
				}
			}
		case <-c.quit:
			break
		}
	}
}

func (c *GorewsClient) checkAndRebuild() {
	err := c.connection.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second))
	if err != nil {
		c.newConnection()
	}
}

func (c *GorewsClient) pingPong() {
	// check connection health every period
	ticker := time.NewTicker(time.Second * 10)
	for {
		select {
		case <-ticker.C:
			c.checkAndRebuild()
		case <-c.quit:
			break
		}
	}
}

func (c *GorewsClient) read() {
	for {
		if !c.connected {
			time.Sleep(time.Second)
			continue
		}
		// read a message from client
		// t, msg, err := c.connection.ReadMessage()
		_, p, err := c.connection.ReadMessage()
		if err != nil {
			logrus.WithError(err).Warn("read error")
			c.connected = false
			time.Sleep(time.Second)
			// time to rebuild the connection
		}

		fmt.Println(string(p))

		//c.Incoming <- msg

	}
}
