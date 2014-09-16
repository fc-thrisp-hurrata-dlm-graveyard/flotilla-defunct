package flotilla

import (
	"fmt"
	"log"
	"net/http"
)

const (
	exceptionHtml = `<!DOCTYPE HTML>
<title>%d %s</title>
<h1>%s</h1>
<p>%s</p>
`
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
		code     int
		message  string
		handlers []HandlerFunc
	}

	// A map of HttpException instances, keyed by status code
	HttpExceptions map[int]*HttpException
)

func defaulthttpexceptions() HttpExceptions {
	httpexceptions := make(HttpExceptions)
	httpexceptions.add(newhttpexception(400, "The browser (or proxy) sent a request that this server could not understand."))
	httpexceptions.add(newhttpexception(401, "The server could not verify that you are authorized to access the URL requested.\nYou either supplied the wrong credentials (e.g. a bad password), or your browser doesn't understand how to supply the credentials required."))
	httpexceptions.add(newhttpexception(403, "You do not have the permission to access the requested resource.\nIt is either read-protected or not readable by the server."))
	httpexceptions.add(newhttpexception(404, "The requested URL was not found on the server. If you entered the URL manually please check your spelling and try again"))
	httpexceptions.add(newhttpexception(405, "The method is not allowed for the requested URL."))
	httpexceptions.add(newhttpexception(418, "This server is a teapot, not a coffee machine"))
	httpexceptions.add(newhttpexception(500, "The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application."))
	httpexceptions.add(newhttpexception(502, "The proxy server received an invalid response from an upstream server."))
	httpexceptions.add(newhttpexception(503, "The server is temporarily unable to service your request due to maintenance downtime or capacity problems. Please try again later."))
	httpexceptions.add(newhttpexception(504, "The connection to an upstream server timed out."))
	httpexceptions.add(newhttpexception(505, "The server does not support the HTTP protocol version used in the request"))
	return httpexceptions
}

func newhttpexception(code int, message string) *HttpException {
	n := &HttpException{code: code, message: message}
	n.handlers = append(n.handlers, n.defaultexceptionpre(), n.defaultexceptionpost())
	return n
}

func (h *HttpException) name() string {
	return http.StatusText(h.code)
}

func (h *HttpException) defaultexceptionpre() HandlerFunc {
	return func(c *Ctx) {
		c.rw.WriteHeader(h.code)
	}
}

func (h *HttpException) defaultexceptionpost() HandlerFunc {
	return func(c *Ctx) {
		if !c.rw.Written() {
			if c.rw.Status() == h.code {
				c.rw.Header().Set("Content-Type", "text/html")
				c.rw.Write(h.format())
			} else {
				c.rw.WriteHeaderNow()
			}
		}
	}
}

func (h *HttpException) format() []byte {
	return []byte(fmt.Sprintf(exceptionHtml, h.code, h.name(), h.name(), h.message))
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
	hs[h.code] = h
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
			c.rw.Header().Set("Content-Type", "text/html")
			c.rw.Write([]byte(fmt.Sprintf(panicHtml, err, err, stack)))
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
