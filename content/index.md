+++
title = "flotilla-index"
+++
| [overview](/flotilla) | // | [quickstart](#quickstart) | // | [extensions](#extensions) | // | [community](#community) |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |

> Flotilla is a basic and extensible web framework for the Go language.


Flotilla is a new project, documentation, testing, and more to come soon.


# Installation<a name="installation"></a>

Flotilla is built around two core dependencies: 

- [engine](http://github.com/thrisp/engine) *routing, multiplexing, net/http basics & interface*

- [djinn](http://github.com/thrisp/djinn/) *templating*


After installing, you can install with `go get -u github.com/thrisp/flotilla`.

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
            b[i] = lucky[rand.Intn(len(lucky))]
        }
        return string(b)
    }

    func Display(f *flotilla.R) {
        ret := fmt.Sprintf("Your lucky number is: %s", randSeq(20))
        f.ServeData(200, []byte(ret))
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
        fl := Build()
        go fl.Run(":8080")
        <-quit
    }

go run main.go & visit: http://localhost:8080/quick/whatsmyluckynumber

# Extensions<a name="extensions"></a>

One of the primary goals of creating Flotilla is a pool of extensions, middleware, and packages that that promote code reuse and the prevention of wasted effort in recreating everything from net/http whenever you want to do something requiring a web framework in Go(unless thats your thing). There are no standard extensions as of right now, but examples for creating extension can be found [here](https://github.com/thrisp/flotilla_skeleton), as well your imagination in reading the Flotilla source code. 

# Community<a name="community"></a>

Found a bug? Have a good idea for improving Flotilla?  Your participation and input is welcome. Visit the [Github page](https://github.com/thrisp/flotilla) to review code, fork & explore, or file an issue. If you're interested in chatting with fellow developers, visit the IRC channel at #flotilla on irc.freenode.net. Helpful links and discussion can be found at [/r/flotilla](http://reddit.com/r/flotillaaa) on [reddit](http://reddit.com). 
