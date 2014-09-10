package flotilla

import (
	"fmt"
	"log"
	"net/http"
)

const (
	panicHtml = `<html>
<head><title>Flotilla Page Error: %s</title>
<style type="text/css">
html, body {
font-family: "Roboto", sans-serif;
color: #333333;
background-color: blue;
margin: 0px;
}
h1 {
color: #d04526;
background-color: #ffffff;
padding: 20px;
border-bottom: 1px dashed #2b3848;
}
pre {
margin: 20px;
padding: 20px;
border: 2px solid #2b3848;
background-color: #ffffff;
}
</style>
</head><body>
<h1>Flotilla Error</h1>
<pre style="font-weight: bold;">%s</pre>
<pre>%s</pre>
</body>
</html>`
)

type (
	// Status code, message, and handlers for a http exception.
	HttpException struct {
		statuscode int
		message    string
		handlers   []HandlerFunc
	}

	// A map of HttpException instances, keyed by status code
	HttpExceptions map[int]*HttpException
)

func defaulthttpexceptions() HttpExceptions {
	httpexceptions := make(HttpExceptions)
	httpexceptions.add(newhttpexception(400, "Bad Request"))
	httpexceptions.add(newhttpexception(401, "Unauthorized"))
	httpexceptions.add(newhttpexception(403, "Forbidden"))
	httpexceptions.add(newhttpexception(404, "Page Not Found"))
	httpexceptions.add(newhttpexception(405, "Method Not Allowed"))
	httpexceptions.add(newhttpexception(500, "Internal Server Error"))
	httpexceptions.add(newhttpexception(502, "Bad Gateway"))
	httpexceptions.add(newhttpexception(503, "Service Unavailable"))
	httpexceptions.add(newhttpexception(504, "Gateway Timeout"))
	return httpexceptions
}

func newhttpexception(statuscode int, message string) *HttpException {
	n := &HttpException{statuscode: statuscode, message: message}
	n.handlers = append(n.handlers, n.defaultexceptionpre(), n.defaultexceptionpost())
	return n
}

func (h *HttpException) defaultexceptionpre() HandlerFunc {
	return func(c *Ctx) {
		c.rw.WriteHeader(h.statuscode)
	}
}

func (h *HttpException) defaultexceptionpost() HandlerFunc {
	return func(c *Ctx) {
		if !c.rw.Written() {
			if c.rw.Status() == h.statuscode {
				c.ServeData(h.statuscode, "text/plain", h.format())
			} else {
				c.rw.WriteHeaderNow()
			}
		}
	}
}

func (h *HttpException) format() []byte {
	return []byte(fmt.Sprintf("%d: %s", h.statuscode, h.message))
}

func (h *HttpException) updatehandlers(handlers ...HandlerFunc) {
	s := len(h.handlers) + len(handlers)
	newh := make([]HandlerFunc, 0, s)
	newh = append(newh, h.handlers[0])
	if len(h.handlers) > 2 {
		newh = append(newh, h.handlers[1:(len(h.handlers)-2)]...)
	}
	newh = append(newh, handlers...)
	newh = append(newh, h.handlers[len(h.handlers)-1])
	h.handlers = newh
}

func (hs HttpExceptions) add(h *HttpException) {
	hs[h.statuscode] = h
}

// A handler for engine.router NotFound handler
func (engine *Engine) handler404(w http.ResponseWriter, req *http.Request) {
	e := engine.HttpExceptions[404]
	c := engine.getCtx(w, req, nil, engine.combineHandlers(e.handlers))
	c.Next()
	engine.cache.Put(c)
}

// A handler for engine.router Panic handler
func (engine *Engine) handler500(w http.ResponseWriter, req *http.Request, err interface{}) {
	e := engine.HttpExceptions[500]
	e.updatehandlers(func(c *Ctx) {
		stack := stack(3)
		log.Printf("\n---------------------\nInternal Server Error\n---------------------\n%s\n---------------------\n%s\n---------------------\n", err, stack)
		switch engine.Env.Mode {
		case devmode:
			servePanic := fmt.Sprintf(panicHtml, err, err, stack)
			c.ServeData(500, "text/html", []byte(servePanic))
		}
	})
	c := engine.getCtx(w, req, nil, engine.combineHandlers(e.handlers))
	c.Next()
	engine.cache.Put(c)
}

// ExceptionHandler updates an existing HttpException, or if non-existent, creates
// a new one, with the provided integer status code, message, and HandlerFuncs
func (engine *Engine) ExceptionHandler(i int, message string, handlers ...HandlerFunc) {
	if _, ok := engine.HttpExceptions[i]; !ok {
		engine.HttpExceptions.add(newhttpexception(i, ""))
	}
	e := engine.HttpExceptions[i]
	if message != "" {
		e.message = message
	}
	e.updatehandlers(handlers...)
}

// UseExceptionHandler adds the provided HandlerFuncs to the given integer
// HttpException in engine.HttpExceptions.
func (engine *Engine) UseExceptionHandler(i int, handlers ...HandlerFunc) {
	if e, ok := engine.HttpExceptions[i]; ok {
		e.updatehandlers(handlers...)
	}
}
