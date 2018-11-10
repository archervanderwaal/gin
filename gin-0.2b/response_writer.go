package gin

import (
	"net/http"
)

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
type (
	// 增强ResponseWriter接口
	ResponseWriter interface {
		http.ResponseWriter
		Status() int
		Written() bool

		// private
		reset(http.ResponseWriter)
		setStatus(int)
	}

	responseWriter struct {
		http.ResponseWriter
		status  int
		written bool
	}
)

// 重置
func (w *responseWriter) reset(writer http.ResponseWriter) {
	w.ResponseWriter = writer
	w.status = 0
	w.written = false
}

// 设置status
func (w *responseWriter) setStatus(code int) {
	w.status = code
}

// 写入响应码
func (w *responseWriter) WriteHeader(code int) {
	w.status = code
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}

// 获取status
func (w *responseWriter) Status() int {
	return w.status
}

// 是否可写
func (w *responseWriter) Written() bool {
	return w.written
}
