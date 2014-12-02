+++
title = "start"
draft = "false"
+++
## Getting Started

# Installation<a name="installation"></a>

Flotilla can be installed with `go get -u github.com/thrisp/flotilla`.

Although designed as close to the standard library as possible, Flotilla is built around two core dependencies: 

- [engine](http://github.com/thrisp/engine) *routing, multiplexing, net/http basics & interface*

- [djinn](http://github.com/thrisp/djinn/) *templating*


# Example<a name="example"></a>

A simple example application using Flotilla.


main.go

    package main

    import (
        "math/rand"
        "os"
        "os/signal"

        "github.com/thrisp/flotilla"
    )

    var lucky = []rune("1234567890")

    func randSeq(n int) string {
        b := make([]rune, n)
        for i := range b {
            b[i] = lucky[rand.Intn(len(lucky))]
        }
        return string(b)
    }

    func Display(f *flotilla.Ctx) {
        ret := fmt.Sprintf("Your lucky number is: %s", randSeq(20))
        f.ServeData(200, []byte(ret))
    }

    func Build() (e *flotilla.App) {
        e = flotilla.New()
        e.GET("/quick/:start", Display)
        return e
    }

    var quit = make(chan bool)

    func init() {
        c := make(chan os.Signal, 1)
        signal.Notify(c, os.Interrupt)
        go func() {
            for _ = range c {
                quit <- true
            }
        }()
    }

    func main() {
        fl := Build()
        go fl.Run(":8080")
        <-quit
    }

go run main.go & visit: http://localhost:8080/quick/whatsmyluckynumber
