package fingy

import (
	"encoding/json"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/websocket"
	"github.com/nicolas-nannoni/fingy-gateway/events"
	"net/url"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	maxMessageSize = 512

	retryInterval = 5 * time.Second

	keepAliveInterval = 5 * time.Second

	// Number of events that can be bufferred before being sent
	eventBufferSize = 10
)

type connection struct {
	ws                      *websocket.Conn
	lastSuccessfulKeepAlive time.Time
	send                    chan []byte

	failed chan bool
}

type fingyClient struct {
	Router      Router
	sendChannel chan events.Event
	DeviceId    string
	ServiceId   string
	FingyHost   string
}

func connectLoop() {

	for {
		err := connectToFingy()
		if err == nil {
			break
		}

		log.Errorf("Error while connecting to Fingy: %s", err)
		log.Infof("Retrying to connect to Fingy in %s...", retryInterval)
		time.Sleep(retryInterval)
	}

}

func connectToFingy() (err error) {

	u := url.URL{Scheme: "ws", Host: F.FingyHost, Path: fmt.Sprintf("/service/%s/device/%s/socket", F.ServiceId, F.DeviceId)}

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	conn := connection{
		ws:   ws,
		send: make(chan []byte, maxMessageSize),

		failed: make(chan bool),
	}

	log.Infof("Connection with Fingy established!")

	go conn.sendLoop()
	go conn.readLoop()
	go conn.writeLoop()
	go conn.keepAliveLoop()

	// Block until failure
	<-conn.failed
	return fmt.Errorf("Connection with Fingy lost")
}

func (c *connection) sendEvent(evt *events.Event) (err error) {

	msg, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("The event %s could not be serialized to JSON: %v", evt, err)
	}
	log.Debugf("Pushing message %s to send queue", msg)
	c.send <- msg

	return
}

func (c *connection) write(mt int, payload []byte) error {
	c.ws.SetWriteDeadline(time.Now().Add(writeWait))
	return c.ws.WriteMessage(mt, payload)
}

func (c *connection) close() {
	c.write(websocket.CloseMessage, []byte{})
	close(c.send)
}

func (c *connection) sendLoop() {
Loop:
	for {
		select {
		case evt := <-F.sendChannel:
			evt.Timestamp = time.Now()
			c.sendEvent(&evt)
		case _, ok := <-c.failed:
			if !ok {
				break Loop
			}
		}

	}
	log.Debugf("Ended send loop")
}

func (c *connection) writeLoop() {
Loop:
	for {
		select {
		case _, ok := <-c.failed:
			if !ok {
				break Loop
			}

		case message, ok := <-c.send:
			if !ok {
				break Loop
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				log.Errorf("Error while sending message to connection: %s", err)
				return
			}
		}
	}
	log.Debugf("Write loop closed")
	c.failed <- true
}

func (c *connection) readLoop() {

	c.ws.SetReadLimit(maxMessageSize)

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Errorf("Error in connection: %s", err)
			}
			break
		}
		dispatchReceivedMessage(message)
	}
	log.Debugf("Read loop closed")
	c.failed <- true
}

func (c *connection) pongHandler(appData string) error {

	log.Debug("Received pong from Fingy!")
	c.lastSuccessfulKeepAlive = time.Now()
	return nil
}

func (c *connection) keepAliveLoop() {

	defer func() {
		log.Error("Keep alive ping set connection in fail mode")
		c.failed <- true
	}()

	c.ws.SetPongHandler(c.pongHandler)
	c.lastSuccessfulKeepAlive = time.Now()

	for {
		if c.lastSuccessfulKeepAlive.Add(writeWait).Before(time.Now()) {
			log.Errorf("No keep alive pong received from Fingy in time: connection is stale")
			c.failed <- true
			return
		}

		if err := c.write(websocket.PingMessage, nil); err != nil {
			log.Errorf("Error while sending keep alive Ping message to connection: %s", err)
			c.failed <- true
			return
		}

		log.Debug("Keep alive ping successfully sent")
		time.Sleep(keepAliveInterval)
	}
}

func dispatchReceivedMessage(msg []byte) {

	log.Debugf("Message received from Fingy %s", msg)
	var evt events.Event
	err := json.Unmarshal(msg, &evt)
	if err != nil {
		log.Print(err)
		return
	}

	log.Debugf("Message parsed from Fingy %v", evt)
	F.Router.Dispatch(&evt)
}
