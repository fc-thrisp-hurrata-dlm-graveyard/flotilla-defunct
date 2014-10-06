package flotilla

import (
	"fmt"
	"log"
	"net/http"
)

const (
	statusHtml = `<!DOCTYPE HTML>
<title>%d %s</title>
<h1>%s</h1>
<p>%s</p>
`
	panicHtml = `<html>
<head><title>PANIC: %s</title>
<style type="text/css">
html, body {
font-family: "Roboto", sans-serif;
color: #333333;
background-color: #ea5343;
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
<h1>PANIC</h1>
<pre style="font-weight: bold;">%s</pre>
<pre>%s</pre>
</body>
</html>
`
)

type (
	// A map keyed by status code for a slice of HandlerFunc
	HttpStatuses map[int][]HandlerFunc
)

func format404() string {
	txt := http.StatusText(404)
	return fmt.Sprintf(statusHtml, 404, txt, txt, "The requested URL was not found on the server. If you entered the URL manually please check your spelling and try again.")
}

func format500() string {
	txt := http.StatusText(500)
	return fmt.Sprintf(statusHtml, 500, txt, txt, "The server encountered an internal error and was unable to complete your request. Either the server is overloaded or there is an error in the application.")
}

func (a *App) default404(w http.ResponseWriter, req *http.Request) {
	// Need an ad-hoc *R for template functions, etc.
	r := a.tmpR(w, req)
	r.rw.WriteHeader(404)
	if handlers, ok := a.HttpStatuses[404]; ok {
		lh := len(handlers)
		for i := 0; i < lh; i++ {
			handlers[i](r)
		}
	} else {
		// We have no engine.Ctx at this point as this is passed down to the
		// engine.router, before a ctx is made. Which works for the router,
		// but is messy for Engine&Flotilla, i.e. works for now but could be better.
		r.rw.Header().Set("Content-Type", "text/html")
		r.rw.Write([]byte(format404()))
	}
}

func (a *App) default500(w http.ResponseWriter, req *http.Request, err interface{}) {
	stack := stack(3)
	log.Printf("\n-----\nPANIC\n-----\nerr: %s\n-----\n%s\n-----\n", err, stack)
	switch a.Env.Mode {
	case devmode, testmode:
		servePanic := fmt.Sprintf(panicHtml, err, err, stack)
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(servePanic))
	case prodmode:
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(format500()))
	}
}
