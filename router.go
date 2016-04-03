package fingy

import (
	"github.com/ant0ine/go-urlrouter"
	"github.com/nicolas-nannoni/fingy-server/events"
	"log"
)

type router struct {
	router urlrouter.Router
}

type Context struct {
	Params map[string]string
}

var Router = router{
	router: urlrouter.Router{
		Routes: []urlrouter.Route{},
	},
}

func (r *router) Entry(path string, dest func(c *Context)) {

	route := urlrouter.Route{
		PathExp: path,
		Dest:    dest,
	}
	r.router.Routes = append(r.router.Routes, route)
	r.router.Start()
}

func (r *router) Dispatch(evt *events.Event) {

	route, params, err := r.router.FindRoute(evt.Path)
	if route == nil {
		log.Printf("Unable to dispatch  %v: no matching route found", evt)
		return
	}
	if err != nil {
		log.Printf("Unable to dispatch event %v: %v", evt, err)
		return
	}

	c := Context{
		Params: params,
	}
	route.Dest.(func(c *Context))(&c)
}
