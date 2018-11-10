package gin

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com

import (
	"fmt"
	"log"
	"time"
)

// 错误日志记录
func ErrorLogger() HandlerFunc {
	return func(c *Context) {
		c.Next()

		if len(c.Errors) > 0 {
			// -1 status code = do not change current one
			c.JSON(-1, c.Errors)
		}
	}
}

// 日志信息记录
func Logger() HandlerFunc {
	return func(c *Context) {

		// Start timer
		t := time.Now()

		// 调用后面的处理函数
		// Process request
		c.Next()

		// 统计处理时间
		// Calculate resolution time
		log.Printf("%s in %v", c.Req.RequestURI, time.Since(t))
		if len(c.Errors) > 0 {
			fmt.Println(c.Errors)
		}
	}
}
