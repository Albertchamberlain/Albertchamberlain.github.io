---
layout: post
title: go原生net库分析
date: 2022-07-31
tags: go设计与实现
---

基于HTTP构建的服务器包括两个端，客户端 (Client) 和服务端 (Server)。HTTP Request从客户端发出，服务端接受到请求后进行处理然后将Response返回给客户端。所以http服务器的工作就在于如何接受来自客户端的请求，对数据加工，并向客户端返回响应。

# server实现
```go
import (
   "fmt"
   "net/http"
)

func HelloHandler(w http.ResponseWriter, r *http.Request) {
   fmt.Fprintf(w, "Hello Amos")
}

func main () {
   http.HandleFunc("/", HelloHandler)
   http.ListenAndServe(":8000", nil)
}
```

这段代码十分简短，流程就是先利用http中的handlefunc在根路由上注册一个handler，然后调用ListenAndServe启动服务器并监听8000端口上的数据。当有请求来的时候，根据之前映射的路由执行对应的Handler函数

`http.ListenAndServe(":8000", nil)`方法中第一个参数前面应该还需要有一个IP地址的，如果不指定会提供一个默认IP，可以跳进源码看看

如下逻辑
```go
if len(ip) == 0 {
		ip = IPv4zero
	}
```
如果不指定IP地址，就会把IPv4zero赋给ip。IPv4zero是一个事先声明好的值，就是0.0.0.0
0.0.0.0指的是本机上的所有IPV4地址，如果一个主机有两个IP地址，192.168.1.1 和 10.1.2.1，并且该主机上的一个服务监听的地址是0.0.0.0,那么通过两个ip地址都能够访问该服务。
```go
// Well-known IPv4 addresses
var (
	IPv4bcast     = IPv4(255, 255, 255, 255) // limited broadcast
	IPv4allsys    = IPv4(224, 0, 0, 1)       // all systems
	IPv4allrouter = IPv4(224, 0, 0, 2)       // all routers
	IPv4zero      = IPv4(0, 0, 0, 0)         // all zeros
)
```

# 路由注册
handleFunc中第一个参数是pattern，其为路由的匹配规则，第二个参数是一个函数。这个函数中又有两个入参http.ResponseWriter和*http.Requests。
```go
// HandleFunc registers the handler function for the given pattern.
func (mux *ServeMux) HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	if handler == nil {
		panic("http: nil handler")
	}
	mux.Handle(pattern, HandlerFunc(handler))
}
// Handle registers the handler for the given pattern
// in the DefaultServeMux.
// The documentation for ServeMux explains how patterns are matched.
func Handle(pattern string, handler Handler) { 
    DefaultServeMux.Handle(pattern, handler) 
}
```
可以发现HandleFunc函数中调用了DefaultServeMux的HandleFunc方法，这里涉及了两种对象，ServeMux和Handler对象

# ServeMux对象 (服务复用器)

其结构体定义如下
```go
type ServeMux struct {
	mu    sync.RWMutex
	m     map[string]muxEntry
	es    []muxEntry // slice of entries sorted from longest to shortest.
	hosts bool       // whether any patterns contain hostnames
}
type muxEntry struct {
	h       Handler
	pattern string
}
```

ServeMux结构体中的字段 mu是一个读写锁，m是一个 map，key 是路由表达式，value 是一个 muxEntry 结构，muxEntry 结构体存储了路由表达式和对应的 handler。字段 m 对应的 map 用于路由的精确匹配 es 字段用于路由的部分匹配

# Handler对象

Handler的定义是一个接口
```go
type Handler interface {
	ServeHTTP(ResponseWriter, *Request) //对HTTP请求与响应行头体进行了封装
}
```
也就是说只要是实现了ServeHTTP方法的结构体，就可以认为其是一个Handler对象
其实刚刚的结构体ServeMux其也实现了该方法
```GO
// ServeHTTP dispatches the request to the handler whose
// pattern most closely matches the request URL.
func (mux *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	if r.RequestURI == "*" {
		if r.ProtoAtLeast(1, 1) {
			w.Header().Set("Connection", "close")
		}
		w.WriteHeader(StatusBadRequest)
		return
	}
	h, _ := mux.Handler(r)
	h.ServeHTTP(w, r)
}
```
也就是说ServeMux也是Handler对象，只不过 ServeMux 的 ServeHTTP 方法直接处理request与response，而是通过路由查找对应的路由处理器 Handler 对象，将其派发给 ServeHTTP 方法。

# 路由注册
因为没有创建ServeMux的话，就会调用DefaultServeMux
```go
func HandleFunc(pattern string, handler func(ResponseWriter, *Request)) {
	DefaultServeMux.HandleFunc(pattern, handler)
}
```

ServeMux的 Handle 方法函数注册路由过程

```go
func (mux *ServeMux) Handle(pattern string, handler Handler) {
    mux.mu.Lock() //获取读写锁
    defer mux.mu.Unlock() //释放读写锁

    if pattern == "" { //如果路由表达式为空，则抛出异常
        panic("http: invalid pattern")
    }
    if handler == nil { //如果handler为空，则抛出异常
        panic("http: nil handler")
    }
  // 路由已经注册过处理器函数，直接panic
    if _, exist := mux.m[pattern]; exist {
        panic("http: multiple registrations for " + pattern)
    }

    if mux.m == nil { //如果路由表为空，则创建一个map
        mux.m = make(map[string]muxEntry)
    }
  // 用路由的pattern和处理函数创建 muxEntry 对象
    e := muxEntry{h: handler, pattern: pattern}
  // 向ServeMux的m 字段增加新的路由匹配规则
    mux.m[pattern] = e
    if pattern[len(pattern)-1] == '/' {
  // 如果路由patterm以'/'结尾，则将对应的muxEntry对象加入到[]muxEntry中，路由长的位于切片的前面
        mux.es = appendSorted(mux.es, e)
    }
    if pattern[0] != '/' { 
        mux.hosts = true
    }
}
```

#启动服务
http.ListenAndServe 方法
```go
func ListenAndServe(addr string, handler Handler) error {
    server := &Server{Addr: addr, Handler: handler} //创建一个Server对象
    return server.ListenAndServe() //启动服务
}

func (srv *Server) ListenAndServe() error {
    if srv.shuttingDown() { //如果服务器正在关闭，则抛出异常
        return ErrServerClosed //服务器已经关闭
    }
    addr := srv.Addr
    if addr == "" {
        addr = ":http"
    }
    ln, err := net.Listen("tcp", addr) //监听端口
    if err != nil {
        return err
    }
    return srv.Serve(tcpKeepAliveListener{ln.(*net.TCPListener)}) //启动服务
}
```
# Server结构体
```go
type Server struct {
    Addr    string //服务器地址 
    Handler Handler //处理器对象  
    TLSConfig *tls.Config //TLS配置对象
    ReadTimeout time.Duration //读取超时时间
    ReadHeaderTimeout time.Duration 
    WriteTimeout time.Duration 
    IdleTimeout time.Duration	
    MaxHeaderBytes int //最大请求头字节数
    TLSNextProto map[string]func(*Server, *tls.Conn, Handler) //TLS协议处理函数
    ConnState func(net.Conn, ConnState) //连接状态回调函数
    ErrorLog *log.Logger //错误日志对象

    disableKeepAlives int32 //禁用keep-alive标记位    
    inShutdown        int32 //关闭标记位     
    nextProtoOnce     sync.Once //协议处理函数初始化标记位 
    nextProtoErr      error //协议处理函数初始化错误     

    mu         sync.Mutex //读写锁
    listeners  map[*net.Listener]struct{} //监听器列表
    activeConn map[*conn]struct{}// 活跃连接
    doneChan   chan struct{} //关闭服务器的信号通道
    onShutdown []func() //关闭服务器回调函数
}
```
1. 在 Server 的 ListenAndServe 方法中，会初始化监听地址 Addr
2. 调用 Listen 方法设置监听。
3. 将监听的 TCP 对象传入 Serve 方法。
4. Serve 方法会接收 Listener 中过来的连接，为每个连接创建一个 goroutine
5. 在 goroutine 中会用路由处理 Handler 对请求进行处理并构建响应。
   
# Serve流程
```go
func (srv *Server) Serve(l net.Listener) error {
   baseCtx := context.Background() 	
   ctx := context.WithValue(baseCtx, ServerContextKey, srv) 
   for {
      rw, e := l.Accept()// 接收 listener 过来的网络连接请求
      c := srv.newConn(rw) 
      c.setState(c.rwc, StateNew) // 将连接放在 Server.activeConn这个 map 中
      go c.serve(ctx)// 创建协程处理请求
   }
}
```
1. 创建一个上下文对象，然后调用 Listener 的 Accept() 接收监听到的网络连接
2. 新的连接建立时调用 Server 的 newConn() 创建新的连接对象，并将连接的状态标志为 StateNew
3. 开启一个 goroutine 处理连接请求。






