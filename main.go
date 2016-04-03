package fingy

import (
	"github.com/nicolas-nannoni/fingy-server/events"
)

var c connection
var F = fingyClient{Send: make(chan events.Event, eventBufferSize)}
var DeviceId string
