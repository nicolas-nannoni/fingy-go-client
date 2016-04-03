# Fingy - Go Device Client Library
Use this library to create a client application communicating with a Fingy service via WebSockets.

**NOTE: this is still a pre-alpha version. Everything can change and documentation is close to non-existing.**

## How to use?
First, download the library:

    go get github.com/nicolas-nannoni/fingy-go-client/

Then, in your app, create a `Router`, add some `Entry` to it and the function that should be called upon match, and then initiate the connection to `Fingy`:

    package main

    import (
        fingy "github.com/nicolas-nannoni/fingy-go-client"
        "log"
    )

    func main() {

        fingy.Router.Entry("/hello", func(c *fingy.Context) {
            log.Print("hello")
        })

        fingy.DeviceId = "abcdef1234"

        go fingy.Run()

        select {}
    }

Any event sent from Fingy server to the device with path `/hello` will match the `Entry` `/hello`.

## Remaining work
* SSL authentication (client side).
* Request-response management.
* Timeout management on messages in buffer.
* Registration/unregistration and error management.
* Built-in aliveness check.