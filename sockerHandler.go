package fingy

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/nicolas-nannoni/fingy-server/events"
	"log"
	"net/url"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	maxMessageSize  = 512
	fingyServerHost = "localhost:8080"

	retryInterval = 5 * time.Second

	// Number of events that can be bufferred before being sent
	eventBufferSize = 10
)

type connection struct {
	ws   *websocket.Conn
	send chan []byte

	failed chan bool
}

type fingyClient struct {
	Send chan events.Event
}

func Run() {

	for {
		err := connectToFingy()
		if err == nil {
			break
		}

		log.Printf("Error while connecting to Fingy: %s", err)
		log.Printf("Retrying to connect to Fingy in %s...", retryInterval)
		time.Sleep(retryInterval)
	}

}

func connectToFingy() (err error) {

	u := url.URL{Scheme: "ws", Host: fingyServerHost, Path: fmt.Sprintf("/device/%s/socket", DeviceId)}

	ws, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}

	conn := connection{
		ws:   ws,
		send: make(chan []byte, maxMessageSize),

		failed: make(chan bool),
	}

	log.Printf("Connection with Fingy established!")

	go conn.sendLoop()
	go conn.readLoop()
	go conn.writeLoop()

	// Block until failure
	<-conn.failed
	close(conn.failed)
	return fmt.Errorf("Connection with Fingy lost")
}

func (c *connection) sendEvent(evt *events.Event) (err error) {

	evt.PrepareForSend()
	err = evt.Verify()
	if err != nil {
		return err
	}

	msg, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("The event %s could not be serialized to JSON: %v", evt, err)
	}
	log.Printf("Pushing message %s to send queue", msg)
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
		case evt := <-F.Send:
			evt.Timestamp = time.Now()
			c.sendEvent(&evt)
		case _, ok := <-c.failed:
			if !ok {
				break Loop
			}
		}

	}
	log.Printf("Ended send loop")
}

func (c *connection) writeLoop() {
Loop:
	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				break Loop
			}
			if err := c.write(websocket.TextMessage, message); err != nil {
				log.Fatalf("Error while sending message to connection %s", c)
				return
			}
		}
	}
	log.Printf("Write loop closed %s", c)
	c.failed <- true
}

func (c *connection) readLoop() {

	c.ws.SetReadLimit(maxMessageSize)

	for {
		_, message, err := c.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				log.Printf("Error in connection %s: %v", c, err)
			}
			break
		}
		dispatchReceivedMessage(message)
	}
	log.Printf("Read loop closed %s", c)
	c.failed <- true
}

func dispatchReceivedMessage(msg []byte) {

	log.Printf("Message received from Fingy %s", msg)
	var evt events.Event
	err := json.Unmarshal(msg, &evt)
	if err != nil {
		log.Print(err)
		return
	}

	log.Printf("Message parsed from Fingy %v", evt)
	Router.Dispatch(&evt)
}
