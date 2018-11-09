package gin

// gin 框架源码阅读笔记
// date: 2018/11/08
// author: archer vanderwaal 一北@archer.vanderwaal@gmail.com
import (
	"encoding/json"
	"encoding/xml"
	"github.com/julienschmidt/httprouter"
	"html/template"
	"log"
	"math"
	"path"
	"net/http"
)

const (
	AbortIndex = math.MaxInt8 / 2
)

type (
	// 定义处理器函数
	HandlerFunc func(*Context)

	//
	H map[string]interface{}

	// 用于收集一个http请求内部发生的错误信息
	// Used internally to collect a error ocurred during a http request.
	ErrorMsg struct {
		Message string      `json:"msg"`
		Meta    interface{} `json:"meta"`
	}

	// Context是gin当中最为重要的一部分, 它用于在中间件当中传递变量, 管理流程, 例如验证请求的JSON, 并返回.
	// Context is the most important part of gin. It allows us to pass variables between middleware,
	// manage the flow, validate the JSON of a request and render a JSON response for example.
	Context struct {
		// 具体请求
		Req      *http.Request
		// 响应接口
		Writer   http.ResponseWriter
		// 附加值
		Keys     map[string]interface{}
		// 错误信息
		Errors   []ErrorMsg
		// 请求的参数, Param数组, Param是一个包含key, value字段的自定义类型
		Params   httprouter.Params
		// 若干个HandlerFunc, type HandlerFunc func(*Context), 主要包括中间件处理函数以及请求处理函数, 请求处理函数位于数组最后一个。
		handlers []HandlerFunc
		// Engine实例, 代表web framework
		engine   *Engine
		// 当前的handler在handlers中的索引值
		index    int8
	}

	// 用于内部管理路由, 一个RouterGroup关联于一个前缀和一系列的handlers(中间件)
	// Used internally to configure router, a RouterGroup is associated with a prefix
	// and an array of handlers (middlewares)
	RouterGroup struct {
		// 中间件
		Handlers []HandlerFunc
		// 前缀
		prefix   string
		// 父RouterGroup
		parent   *RouterGroup
		// Engine
		engine   *Engine
	}

	// 表征web framework
	// Represents the web framework, it wrappers the blazing fast httprouter multiplexer and a list of global middlewares.
	Engine struct {
		// 路由组
		*RouterGroup
		// 处理404的函数
		handlers404   []HandlerFunc
		// http router
		router        *httprouter.Router
		// 模板
		HTMLTemplates *template.Template
	}
)

// 创建一个不包含任何中间件的一个engine
// Returns a new blank Engine instance without any middleware attached.
// The most basic configuration
func New() *Engine {
	engine := &Engine{}
	engine.RouterGroup = &RouterGroup{nil, "", nil, engine}
	engine.router = httprouter.New()
	// NotFound是一个http.Handler接口, 包含ServeHTTP(ResponseWriter, *Request)方法, handle404是一个
	// func(ResponseWriter, *Request)类型, 没有实现ServeHTTP(ResponseWriter, *Request)方法, 故不可赋值给
	// engine.router.NotFound, engine是一个实现了ServeHTTP(ResponseWriter, *Request)方法的类型, 如果以下代码
	// 修改为engine.router.NotFound = engine, 会导致404的请求无限递归致Stack Overflow, 原因是engine实现的ServeHTTP内部
	// 直接调用httprouter的router.ServeHTTP, 查看router.ServeHTTP方法最后一行
	// 	Handle 404
	//	if r.NotFound != nil {
	//		r.NotFound.ServeHTTP(w, req)
	//	} else {
	//		http.NotFound(w, req)
	//	}
	// 可以发现又回到了engine.ServeHTTP方法！导致栈溢出, 故注释如下代码

	//engine.router.NotFound = engine.handle404
	return engine
}

// 创建一个默认的Engine, 包含两个默认的中间件处理函数, Recovery和Logger
// Returns a Engine instance with the Logger and Recovery already attached.
func Default() *Engine {
	engine := New()
	engine.Use(Recovery(), Logger())
	return engine
}

// 加载HTML模板
func (engine *Engine) LoadHTMLTemplates(pattern string) {
	engine.HTMLTemplates = template.Must(template.ParseGlob(pattern))
}

// 设置404处理函数集
// Adds handlers for NotFound. It return a 404 code by default.
func (engine *Engine) NotFound404(handlers ...HandlerFunc) {
	engine.handlers404 = handlers
}

// 处理404
func (engine *Engine) handle404(w http.ResponseWriter, req *http.Request) {
	handlers := engine.combineHandlers(engine.handlers404)
	c := engine.createContext(w, req, nil, handlers)
	if engine.handlers404 == nil {
		http.NotFound(c.Writer, c.Req)
	} else {
		c.Writer.WriteHeader(404)
	}

	c.Next()
}

// 保存文件
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

// 处理http
// ServeHTTP makes the router implement the http.Handler interface.
func (engine *Engine) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	engine.router.ServeHTTP(w, req)
}

// 启动
func (engine *Engine) Run(addr string) {
	http.ListenAndServe(addr, engine)
}

/************************************/
/********** ROUTES GROUPING *********/
/************************************/

// 新建一个Context, 用来传递这个路由组的数据
func (group *RouterGroup) createContext(w http.ResponseWriter, req *http.Request, params httprouter.Params, handlers []HandlerFunc) *Context {
	return &Context{
		Writer:   w,
		Req:      req,
		index:    -1,
		engine:   group.engine,
		Params:   params,
		handlers: handlers,
	}
}

// 添加一些中间件到这个路由组
// Adds middlewares to the group, see example code in github.
func (group *RouterGroup) Use(middlewares ...HandlerFunc) {
	group.Handlers = append(group.Handlers, middlewares...)
}

// 创建一个新的路由组, 一个路由组应该共享相同的前缀或者相同的中间件, 比如鉴权之类的api可以使用鉴权路由组中的鉴权中间件
// Greates a new router group. You should create add all the routes that share that have common middlwares or same path prefix.
// For example, all the routes that use a common middlware for authorization could be grouped.
func (group *RouterGroup) Group(component string, handlers ...HandlerFunc) *RouterGroup {
	prefix := path.Join(group.prefix, component)
	return &RouterGroup{
		// 添加中间件, 继承自父路由组
		Handlers: group.combineHandlers(handlers),
		parent:   group,
		prefix:   prefix,
		engine:   group.engine,
	}
}

// 注册一个路由
// Handle registers a new request handle and middlewares with the given path and method.
// The last handler should be the real handler, the other ones should be middlewares that can and should be shared among different routes.
// See the example code in github.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (group *RouterGroup) Handle(method, p string, handlers []HandlerFunc) {
	// 参数说明, method为http request method, p为请求url, handlers为处理函数
	// 完整的请求url为路由组前缀+注册路由设置的url
	p = path.Join(group.prefix, p)
	// 处理函数为路由组中间件处理函数加特定的请求处理函数
	handlers = group.combineHandlers(handlers)
	// 注册
	group.engine.router.Handle(method, p, func(w http.ResponseWriter, req *http.Request, params httprouter.Params) {
		group.createContext(w, req, params, handlers).Next()
	})
}

// 快捷函数, 快速注册一个method=POST的路由
// POST is a shortcut for router.Handle("POST", path, handle)
func (group *RouterGroup) POST(path string, handlers ...HandlerFunc) {
	group.Handle("POST", path, handlers)
}

// 快捷函数, 快速注册一个method=GET的路由
// GET is a shortcut for router.Handle("GET", path, handle)
func (group *RouterGroup) GET(path string, handlers ...HandlerFunc) {
	group.Handle("GET", path, handlers)
}

// 快捷函数, 快速注册一个method=DELETE的路由
// DELETE is a shortcut for router.Handle("DELETE", path, handle)
func (group *RouterGroup) DELETE(path string, handlers ...HandlerFunc) {
	group.Handle("DELETE", path, handlers)
}

// 快捷函数, 快速注册一个method=PATCH的路由
// PATCH is a shortcut for router.Handle("PATCH", path, handle)
func (group *RouterGroup) PATCH(path string, handlers ...HandlerFunc) {
	group.Handle("PATCH", path, handlers)
}

// 快捷函数, 快速注册一个method=PUT的路由
// PUT is a shortcut for router.Handle("PUT", path, handle)
func (group *RouterGroup) PUT(path string, handlers ...HandlerFunc) {
	group.Handle("PUT", path, handlers)
}

// 返回一个路由组中间件处理函数加指定的处理函数的集合
func (group *RouterGroup) combineHandlers(handlers []HandlerFunc) []HandlerFunc {
	s := len(group.Handlers) + len(handlers)
	h := make([]HandlerFunc, 0, s)
	h = append(h, group.Handlers...)
	h = append(h, handlers...)
	return h
}

/************************************/
/****** FLOW AND ERROR MANAGEMENT****/
/************************************/

// 下一个应该被调用的中间件
// Next should be used only in the middlewares.
// It executes the pending handlers in the chain inside the calling handler.
// See example in github.
func (c *Context) Next() {
	c.index++
	s := int8(len(c.handlers))
	// 执行具体的handler
	for ; c.index < s; c.index++ {
		c.handlers[c.index](c)
	}
}

// 终止请求, 例如在授权不通过的时候, 直接返回http status 401的响应
// Forces the system to do not continue calling the pending handlers.
// For example, the first handler checks if the request is authorized. If it's not, context.Abort(401) should be called.
// The rest of pending handlers would never be called for that request.
func (c *Context) Abort(code int) {
	// 容易产生误会的一个方法, 参数值为http status
	c.Writer.WriteHeader(code)
	c.index = AbortIndex
}

// 错误处理
// Fail is the same than Abort plus an error message.
// Calling `context.Fail(500, err)` is equivalent to:
// ```
// context.Error("Operation aborted", err)
// context.Abort(500)
// ```
func (c *Context) Fail(code int, err error) {
	c.Error(err, "Operation aborted")
	c.Abort(code)
}

// 当前发生错误, 追加到错误信息集合中
// Attachs an error to the current context. The error is pushed to a list of errors.
// It's a gooc idea to call Error for each error ocurred during the resolution of a request.
// A middleware can be used to collect all the errors and push them to a database together, print a log, or append it in the HTTP response.
func (c *Context) Error(err error, meta interface{}) {
	c.Errors = append(c.Errors, ErrorMsg{
		Message: err.Error(),
		Meta:    meta,
	})
}

/************************************/
/******** METADATA MANAGEMENT********/
/************************************/

// 设置一个key/value数据到特定的context上, 使用懒初始化keys！
// Sets a new pair key/value just for the specefied context.
// It also lazy initializes the hashmap
func (c *Context) Set(key string, item interface{}) {
	if c.Keys == nil {
		// 懒初始化
		c.Keys = make(map[string]interface{})
	}
	c.Keys[key] = item
}

// 根据key获取keys的value, 如果不存在会panic
// Returns the value for the given key.
// It panics if the value doesn't exist.
func (c *Context) Get(key string) interface{} {
	var ok bool
	var item interface{}
	if c.Keys != nil {
		item, ok = c.Keys[key]
	} else {
		item, ok = nil, false
	}
	// item == nil 不存在key对应的value
	if !ok || item == nil {
		log.Panicf("Key %s doesn't exist", key)
	}
	return item
}

/************************************/
/******** ENCOGING MANAGEMENT********/
/************************************/

// 同下面的ParseBody, 只不过EnsureBody发现json不合法, 则会直接响应http status 400
// Like ParseBody() but this method also writes a 400 error if the json is not valid.
func (c *Context) EnsureBody(item interface{}) bool {
	if err := c.ParseBody(item); err != nil {
		c.Fail(400, err)
		return false
	}
	return true
}

// 请求体作为json进行解析
// Parses the body content as a JSON input. It decodes the json payload into the struct specified as a pointer.
func (c *Context) ParseBody(item interface{}) error {
	decoder := json.NewDecoder(c.Req.Body)
	if err := decoder.Decode(&item); err == nil {
		// 进行下一步验证
		return Validate(c, item)
	} else {
		return err
	}
}

// 序列化一个给定的类型为json格式, 并响应, http status为code参数, 并设置响应头部信息为application/json
// Serializes the given struct as a JSON into the response body in a fast and efficient way.
// It also sets the Content-Type as "application/json"
func (c *Context) JSON(code int, obj interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(c.Writer)
	// Encode方法序列化成功会通过io.Writer接口write, 此处为c.Writer(ResponseWriter)
	if err := encoder.Encode(obj); err != nil {
		c.Error(err, obj)
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 同上, 序列化为XML格式
// Serializes the given struct as a XML into the response body in a fast and efficient way.
// It also sets the Content-Type as "application/xml"
func (c *Context) XML(code int, obj interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "application/xml")
	encoder := xml.NewEncoder(c.Writer)
	if err := encoder.Encode(obj); err != nil {
		c.Error(err, obj)
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 模板引擎渲染HTML
// Renders the HTTP template specified by his file name.
// It also update the HTTP code and sets the Content-Type as "text/html".
// See http://golang.org/doc/articles/wiki/
func (c *Context) HTML(code int, name string, data interface{}) {
	if code >= 0 {
		c.Writer.WriteHeader(code)
	}
	c.Writer.Header().Set("Content-Type", "text/html")
	if err := c.engine.HTMLTemplates.ExecuteTemplate(c.Writer, name, data); err != nil {
		c.Error(err, map[string]interface{}{
			"name": name,
			"data": data,
		})
		http.Error(c.Writer, err.Error(), 500)
	}
}

// 响应String
// Writes the given string into the response body and sets the Content-Type to "text/plain"
func (c *Context) String(code int, msg string) {
	c.Writer.Header().Set("Content-Type", "text/plain")
	c.Writer.WriteHeader(code)
	c.Writer.Write([]byte(msg))
}

// 响应流数据
// Writes some data into the body stream and updates the HTTP code
func (c *Context) Data(code int, data []byte) {
	c.Writer.WriteHeader(code)
	c.Writer.Write(data)
}
