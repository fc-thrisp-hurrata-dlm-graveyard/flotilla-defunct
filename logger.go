package flotilla

import (
	"log"
	"os"
	"time"
)

var (
	green   = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white   = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow  = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red     = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	blue    = string([]byte{27, 91, 57, 55, 59, 52, 52, 109})
	magenta = string([]byte{27, 91, 57, 55, 59, 52, 53, 109})
	cyan    = string([]byte{27, 91, 57, 55, 59, 52, 54, 109})
	reset   = string([]byte{27, 91, 48, 109})
)

func Logger() HandlerFunc {
	stdlogger := log.New(os.Stdout, "", 0)

	return func(r *R) {
		// Start timer
		start := time.Now()

		// Process request
		r.Next()

		req := r.Request

		// save the IP of the requester
		requester := req.Header.Get("X-Real-IP")

		// if the requester-header is empty, check the forwarded-header
		if len(requester) == 0 {
			requester = req.Header.Get("X-Forwarded-For")
		}

		// if the requester is still empty, use the hard-coded address from the socket
		if len(requester) == 0 {
			requester = req.RemoteAddr
		}

		var color string
		code := r.rw.Status()
		switch {
		case code >= 200 && code <= 299:
			color = green
		case code >= 300 && code <= 399:
			color = white
		case code >= 400 && code <= 499:
			color = yellow
		default:
			color = red
		}
		var methodColor string
		method := r.Request.Method
		switch {
		case method == "GET":
			methodColor = blue
		case method == "POST":
			methodColor = cyan
		case method == "PUT":
			methodColor = yellow
		case method == "DELETE":
			methodColor = red
		case method == "PATCH":
			methodColor = green
		case method == "HEAD":
			methodColor = magenta
		case method == "OPTIONS":
			methodColor = white
		}
		end := time.Now()
		latency := end.Sub(start)
		stdlogger.Printf("[FLOTILLA] %v |%s %3d %s| %12v | %s |%s  %s %-7s %s %s",
			end.Format("2006/01/02 - 15:04:05"),
			color, code, reset,
			latency,
			requester,
			methodColor, reset, method,
			r.Request.URL.Path,
			r.Ctx.Errors.String(),
		)
	}
}
