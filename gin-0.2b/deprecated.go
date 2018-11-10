package gin

// gin 框架源码阅读笔记
// date: 2018/11/10
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
// 此文件的方法弃用, 使用binding文件代替
import (
	"net/http"
	"github.com/archervanderwaal/gin/gin-0.2b/binding"
)

// DEPRECATED, use Bind() instead.
// Like ParseBody() but this method also writes a 400 error if the json is not valid.
func (c *Context) EnsureBody(item interface{}) bool {
	return c.Bind(item)
}

// DEPRECATED use bindings directly
// Parses the body content as a JSON input. It decodes the json payload into the struct specified as a pointer.
func (c *Context) ParseBody(item interface{}) error {
	return binding.JSON.Bind(c.Req, item)
}

// DEPRECATED use gin.Static() instead
// ServeFiles serves files from the given file system root.
// The path must end with "/*filepath", files are then served from the local
// path /defined/root/dir/*filepath.
// For example if root is "/etc" and *filepath is "passwd", the local file
// "/etc/passwd" would be served.
// Internally a http.FileServer is used, therefore http.NotFound is used instead
// of the Router's NotFound handler.
// To use the operating system's file system implementation,
// use http.Dir:
//     router.ServeFiles("/src/*filepath", http.Dir("/var/www"))
func (engine *Engine) ServeFiles(path string, root http.FileSystem) {
	engine.router.ServeFiles(path, root)
}
