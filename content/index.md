+++
title = "flotilla-index"
+++

| [overview](/flotilla) | // | [quickstart](#quickstart) | // | [extensions](#extensions) | // | [community](#community) |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |

> Flotilla is a basic and extensible web framework for the Go language.

# Installation<a name="installation"></a>

Flotilla has several dependencies:

- [jingo](http://github.com/thrisp/jingo/) *templating*

- [engine](http://github.com/thrisp/engine) *routing, net/http wrappers, basics*

- [kingpin](http://gopkg.in/alecthomas/kingpin.v1)  *flag parsing*

After installing, you can install with `go get github.com/thrisp/flotilla`.

# Quickstart<a name="quickstart"></a>

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
            b[i] = letters[rand.Intn(len(lucky))]
        }
        return string(b)
    }

    func Display(f *flotilla.R) {
        f.ServeData(200, fmt.Sprintf("Your lucky number is: %s", randSeq(20)))
    }

    func Build() (e *flotilla.App) {
        e = flotilla.Basic()
        e.Use(flotilla.ThirdEye)
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
        c := Build()
        go c.Run(":8080")
        <-quit
    }

go run main.go & visit: http://localhost:8080/quick/whatsmyluckynumber

# Extensions<a name="extensions"></a>

One of the primary goals of creating Flotilla is a pool of extensions, middleware, and packages that that promote code reuse and the wasted effort of recreating everything from net/http whenever you want to do something requiring a web framework in Go(unless thats your thing). There are no standard extensions as of right now, but examples for creating extension can be found [here](https://github.com/thrisp/flotilla_skeleton), as well your imagination in reading the Flotilla source code. 

# Community<a name="community"></a>

Flotilla is a new package and you participation & input is welcome. You can find a start for discussion at [/r/flotilla](http://reddit.com/r/flotillaaa) on reddit.
