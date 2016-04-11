package fingy

import (
	"github.com/ant0ine/go-urlrouter"
	"github.com/nicolas-nannoni/fingy-gateway/events"
)

var F = fingyClient{
	Router: Router{
		Router: urlrouter.Router{Routes: []urlrouter.Route{}},
	},
	sendChannel: make(chan events.Event, eventBufferSize),
}

func (f *fingyClient) Begin() (err error) {

	go connectLoop()

	return

}

func (f *fingyClient) Send(evt events.Event) (err error) {

	evt.ServiceId = f.ServiceId
	evt.PrepareForSend()

	err = evt.Verify()
	if err != nil {
		return err
	}

	F.sendChannel <- evt
	return
}
