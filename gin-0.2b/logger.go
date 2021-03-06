package gin

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
import (
	"log"
	"os"
	"time"
)

func ErrorLogger() HandlerFunc {
	return ErrorLoggerT(ErrorTypeAll)
}

func ErrorLoggerT(typ uint32) HandlerFunc {
	return func(c *Context) {
		c.Next()

		errs := c.Errors.ByType(typ)
		if len(errs) > 0 {
			// -1 status code = do not change current one
			c.JSON(-1, c.Errors)
		}
	}
}

var (
	green  = string([]byte{27, 91, 57, 55, 59, 52, 50, 109})
	white  = string([]byte{27, 91, 57, 48, 59, 52, 55, 109})
	yellow = string([]byte{27, 91, 57, 55, 59, 52, 51, 109})
	red    = string([]byte{27, 91, 57, 55, 59, 52, 49, 109})
	reset  = string([]byte{27, 91, 48, 109})
)

// 相比0.1增加了按照等级分颜色显示
func Logger() HandlerFunc {
	stdlogger := log.New(os.Stdout, "", 0)
	//errlogger := log.New(os.Stderr, "", 0)

	return func(c *Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// save the IP of the requester
		requester := c.Req.Header.Get("X-Real-IP")
		// if the requester-header is empty, check the forwarded-header
		if requester == "" {
			requester = c.Req.Header.Get("X-Forwarded-For")
		}

		// if the requester is still empty, use the hard-coded address from the socket
		if requester == "" {
			requester = c.Req.RemoteAddr
		}

		var color string
		code := c.Writer.Status()
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
		end := time.Now()
		latency := end.Sub(start)
		stdlogger.Printf("[GIN] %v |%s %3d %s| %12v | %s %4s %s\n",
			end.Format("2006/01/02 - 15:04:05"),
			color, c.Writer.Status(), reset,
			latency,
			requester,
			c.Req.Method, c.Req.URL.Path,
		)

		// Calculate resolution time
		if len(c.Errors) > 0 {
			stdlogger.Println(c.Errors.String())
		}
	}
}
