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

type OnConnectedCallback func(*GorewsClient)

type GorewsClient struct {
	Incoming            chan []byte
	Outgoing            chan []byte
	OnConnectedCallback OnConnectedCallback
	sending             []byte
	connected           bool
	quit                chan bool
	quited              bool
	Url                 *url.URL
	ReqHeader           http.Header
	Dialer              *websocket.Dialer
	connection          *websocket.Conn
	handshakeTimeout    time.Duration
	writeTimeout        time.Duration
	readTimeout         time.Duration
}

func NewGorewsClient(onConnectedCallback OnConnectedCallback) *GorewsClient {
	return &GorewsClient{
		Incoming:            make(chan []byte, ChanBufferSize),
		Outgoing:            make(chan []byte, ChanBufferSize),
		OnConnectedCallback: onConnectedCallback,
		quit:                make(chan bool),
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
		if c.OnConnectedCallback != nil {
			c.OnConnectedCallback(c)
		}
		return
	}
}

func (c *GorewsClient) Stop() {
	c.quited = true
	close(c.quit)
	close(c.Outgoing)
	close(c.Incoming)
	if c.connected {
		c.connected = false
		_ = c.connection.Close()
	}
}

func (c *GorewsClient) IsConnected() bool {
	return c.connected
}

func (c *GorewsClient) write() {
	for {
		// consume a message from the channel
		select {
		case <-c.quit:
			break
		case msg := <-c.Outgoing:
			// send until success
			c.sending = msg
			for {
				select {
				case <-c.quit:
					logrus.Info("return write")
					return
				default:
				}

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
			return
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
			select {
			case <-c.quit:
				logrus.Info("return read")
				return
			default:
			}
		}

		fmt.Println(string(p))

		select {
		case <-c.quit:
			logrus.Info("return read2")
			return
		case c.Incoming <- p:
			continue
		}
	}
}
