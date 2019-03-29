# beego执行流程

## 初始化
当在运行main函数中的beego.Run方法后首先执行initBeforeHTTPRun函数进行初始化

initBeforeHTTPRun将会调用AddAPPStartHook注册钩子函数，注册的钩子函数如下

### registerMime函数
registerMime通过AddExtensionType函数注册文件后缀名与Content-Type之间的映射关系(MIME Type)，例如.jpeg文件的Content-Type为image/jpeg，相关映射放在beego/mime.go文件中


```go
func registerMime() error {
	for k, v := range mimemaps {
		mime.AddExtensionType(k, v)
	}
	return nil
}
```
### registerDefaultErrorHandler函数
registerDefaultErrorHandler函数用于注册40x以及50x错误的处理相关的handler（401,402,403,404,405,417,422,500,501,502,503,504）

```go
func registerDefaultErrorHandler() error {
	m := map[string]func(http.ResponseWriter, *http.Request){
		"401": unauthorized,
		"402": paymentRequired,
		"403": forbidden,
		"404": notFound,
		"405": methodNotAllowed,
		"500": internalServerError,
		"501": notImplemented,
		"502": badGateway,
		"503": serviceUnavailable,
		"504": gatewayTimeout,
		"417": invalidxsrf,
		"422": missingxsrf,
	}
	for e, h := range m {
		if _, ok := ErrorMaps[e]; !ok {
			ErrorHandler(e, h)
		}
	}
	return nil
}
```

### registerSession函数
如果在在app.conf中启用session将会启用session功能
```go
func registerSession() error {
	if BConfig.WebConfig.Session.SessionOn {
		var err error
		sessionConfig := AppConfig.String("sessionConfig")
		conf := new(session.ManagerConfig)
		if sessionConfig == "" {
			conf.CookieName = BConfig.WebConfig.Session.SessionName
			conf.EnableSetCookie = BConfig.WebConfig.Session.SessionAutoSetCookie
			conf.Gclifetime = BConfig.WebConfig.Session.SessionGCMaxLifetime
			conf.Secure = BConfig.Listen.EnableHTTPS
			conf.CookieLifeTime = BConfig.WebConfig.Session.SessionCookieLifeTime
			conf.ProviderConfig = filepath.ToSlash(BConfig.WebConfig.Session.SessionProviderConfig)
			conf.DisableHTTPOnly = BConfig.WebConfig.Session.SessionDisableHTTPOnly
			conf.Domain = BConfig.WebConfig.Session.SessionDomain
			conf.EnableSidInHTTPHeader = BConfig.WebConfig.Session.SessionEnableSidInHTTPHeader
			conf.SessionNameInHTTPHeader = BConfig.WebConfig.Session.SessionNameInHTTPHeader
			conf.EnableSidInURLQuery = BConfig.WebConfig.Session.SessionEnableSidInURLQuery
		} else {
			if err = json.Unmarshal([]byte(sessionConfig), conf); err != nil {
				return err
			}
		}
		if GlobalSessions, err = session.NewManager(BConfig.WebConfig.Session.SessionProvider, conf); err != nil {
			return err
		}
		go GlobalSessions.GC()
	}
	return nil
}
```

### registerTemplate函数
registerTemplate函数使用AddViewPath来配置View文件路径，如果在beego.Run()调用将会panic
```go
func registerTemplate() error {
	defer lockViewPaths()
	if err := AddViewPath(BConfig.WebConfig.ViewsPath); err != nil {
		if BConfig.RunMode == DEV {
			logs.Warn(err)
		}
		return err
	}
	return nil
}

// AddViewPath adds a new path to the supported view paths.
//Can later be used by setting a controller ViewPath to this folder
func AddViewPath(viewPath string) error {
	if beeViewPathTemplateLocked {
		if _, exist := beeViewPathTemplates[viewPath]; exist {
			return nil //Ignore if viewpath already exists
		}
		panic("Can not add new view paths after beego.Run()")
	}
	beeViewPathTemplates[viewPath] = make(map[string]*template.Template)
	return BuildTemplate(viewPath)
}
```

### registerAdmin函数
如果在配置文件中指定EnableAdmin为True，registerAdmin将会开启一个goroutine，BeeAdminApp是使用admin module的一个默认的主要用于beego系统的监控情况

```go
func registerAdmin() error {
	if BConfig.Listen.EnableAdmin {
		go beeAdminApp.Run()
	}
	return nil
}
```
通过源码可以看到，beeAdminApp注册了几个path，在使用时避免使用重复的path

```go
func init() {
	beeAdminApp = &adminApp{
		routers: make(map[string]http.HandlerFunc),
	}
	beeAdminApp.Route("/", adminIndex)
	beeAdminApp.Route("/qps", qpsIndex)
	beeAdminApp.Route("/prof", profIndex)
	beeAdminApp.Route("/healthcheck", healthcheck)
	beeAdminApp.Route("/task", taskStatus)
	beeAdminApp.Route("/listconf", listConf)
	FilterMonitorFunc = func(string, string, time.Duration, string, int) bool { return true }
}

```

### registerGzip函数

如果在配置文件中启用EnableGzip将会使用context的InitGzip函数来初始化Gzip相关配置(使用默认配置)
```go
func registerGzip() error {
	if BConfig.EnableGzip {
		context.InitGzip(
			AppConfig.DefaultInt("gzipMinLength", -1),
			AppConfig.DefaultInt("gzipCompressLevel", -1),
			AppConfig.DefaultStrings("includedMethods", []string{"GET"}),
		)
	}
	return nil
}
```
其中registerGzip，registerAdmin这两个钩子函数需要通过配置文件才能生效

beego.Run中可以配置参数，主要是监听地址以及端口例如":8080"或者"127.0.0.1:8080"

配置的地址和端口将会保存到config.go文件中的Config结构体中的Listen结构体中的HTTPAddr字段和HTTPPort字段，所注册的地址和端口也会保存在Domains字段中

一切完成后将调用BeeApp.Run()来启用服务

## 启动服务

### 准备工作
启动服务前需要做一些准备工作,例如需要实例化BeeApp

```go
var (
	// BeeApp is an application instance
	BeeApp *App
)

func init() {
	// create beego application
	BeeApp = NewApp()
}

func NewApp() *App {
	cr := NewControllerRegister()
	app := &App{Handlers: cr, Server: &http.Server{}}
	return app
}
```

其中主要的结构为App,相关定义如下
```go
type App struct {
	Handlers *ControllerRegister
	Server   *http.Server
}
```
App结构中包含了实现了ServeMux接口的ControllerRegister，ServeMux是一个多路复用器具体内容可以参考[server](note/web/server.md)中的处理函数一节

可以看到NewApp中有一个NewControllerRegister()函数调用，该函数用于初始化ControllerRegister，以及Context结构

```go
func NewControllerRegister() *ControllerRegister {
	cr := &ControllerRegister{
		routers:  make(map[string]*Tree),
		policies: make(map[string]*Tree),
	}
	cr.pool.New = func() interface{} {
		return beecontext.NewContext()
	}
	return cr
}

```
准备工作完成后便可以启动服务

### 启动服务
启动服务的流程有些复杂，首先BeeApp.Run()函数中可以传递几个MiddleWare在

```go
type MiddleWare func(http.Handler) http.Handler
```

启动服务主要流程
+ 如果在配置文件中启用了EnableFcgi则会开启CGI服务
+ 设置App.Server中的Handler字段，handler已经在App.Handlers初始化过，如果有设置MiddleWare则使用MiddleWare调用结果设置Handler字段，设置Log，ReadTimeout，WriteTimeout等字段（根据配置文件中的内容或者使用默认的）
+ 如果在配置文件中启用了GracefulMod 使用GraceFul启动http sever
+ 启用了HTTPS或者MutualHTTPS则使用HTTPS或者MutialHttps启动server
+ 如果启用HTTP则按照HTTP方式启动server

HTTP，HTTPS，CGI，Graceful都会开启一个单独的写成去完成内容完成后则会向endRunning发送true表示已经完成相应操作（如果设置了多个启动方式，只要有完成了将会通知主goroutine）一般情况下启动出错，或者关闭服务器，服务器出错等都会先endRunning发送true



### CGI启动

```go
// run cgi server
	if BConfig.Listen.EnableFcgi {
		if BConfig.Listen.EnableStdIo {
			if err = fcgi.Serve(nil, app.Handlers); err == nil { // standard I/O
				logs.Info("Use FCGI via standard I/O")
			} else {
				logs.Critical("Cannot use FCGI via standard I/O", err)
			}
			return
		}
		if BConfig.Listen.HTTPPort == 0 {
			// remove the Socket file before start
			if utils.FileExists(addr) {
				os.Remove(addr)
			}
			l, err = net.Listen("unix", addr)
		} else {
			l, err = net.Listen("tcp", addr)
		}
		if err != nil {
			logs.Critical("Listen: ", err)
		}
		if err = fcgi.Serve(l, app.Handlers); err != nil {
			logs.Critical("fcgi.Serve: ", err)
		}
		return
	}
```

### Graceful启动

Graceful启动也会分为HTTPS启动或者HTTP启动

#### HTTPS启动

启动步骤：
1. 根据用户设置或者默认配置从设置地址以及端口
2. 使用grace包中的NewServer方式启动服务器，根据配置设置读写超时时间
3. 如果设置了MutualHTTPS则会调用ListenAndServeMutualTLS，如果没有设置MutualHTTPS但是设置了AutoTLS，则会autocert.Manager的GetCertificate来生成Cert文件，如果没有设置则会在配置文件找找HTTPSCertFile，HTTPSKeyFile相关设置来启动
4. 最后将采用tcp4来启动HTTP server


```go
if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
    go func() {
        time.Sleep(1000 * time.Microsecond)
        if BConfig.Listen.HTTPSPort != 0 {
            httpsAddr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
            app.Server.Addr = httpsAddr
        }
        server := grace.NewServer(httpsAddr, app.Handlers)
        server.Server.ReadTimeout = app.Server.ReadTimeout
        server.Server.WriteTimeout = app.Server.WriteTimeout
        if BConfig.Listen.EnableMutualHTTPS {
            if err := server.ListenAndServeMutualTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile, BConfig.Listen.TrustCaFile); err != nil {
                logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
                time.Sleep(100 * time.Microsecond)
                endRunning <- true
            }
        } else {
            if BConfig.Listen.AutoTLS {
                m := autocert.Manager{
                    Prompt:     autocert.AcceptTOS,
                    HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
                    Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
                }
                app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
                BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
            }
            if err := server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
                logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
                time.Sleep(100 * time.Microsecond)
                endRunning <- true
            }
        }
    }()
}

```

#### HTTP启动


```go
	if BConfig.Listen.Graceful {
		httpsAddr := BConfig.Listen.HTTPSAddr
		app.Server.Addr = httpsAddr
		if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
			go func() {
				time.Sleep(1000 * time.Microsecond)
				if BConfig.Listen.HTTPSPort != 0 {
					httpsAddr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
					app.Server.Addr = httpsAddr
				}
				server := grace.NewServer(httpsAddr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.EnableMutualHTTPS {
					if err := server.ListenAndServeMutualTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile, BConfig.Listen.TrustCaFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				} else {
					if BConfig.Listen.AutoTLS {
						m := autocert.Manager{
							Prompt:     autocert.AcceptTOS,
							HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
							Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
						}
						app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
						BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
					}
					if err := server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}
			}()
		}
		if BConfig.Listen.EnableHTTP {
			go func() {
				server := grace.NewServer(addr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.ListenTCP4 {
					server.Network = "tcp4"
				}
				if err := server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err, fmt.Sprintf("%d", os.Getpid()))
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}()
		}
		<-endRunning
		return
	}
```

### HTTPS启动


### HTTP启动


Run方法内容如下
```go
// Run beego application.
func (app *App) Run(mws ...MiddleWare) {
	addr := BConfig.Listen.HTTPAddr

	if BConfig.Listen.HTTPPort != 0 {
		addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPAddr, BConfig.Listen.HTTPPort)
	}

	var (
		err        error
		l          net.Listener
		endRunning = make(chan bool, 1)
	)

	// run cgi server
	if BConfig.Listen.EnableFcgi {
		if BConfig.Listen.EnableStdIo {
			if err = fcgi.Serve(nil, app.Handlers); err == nil { // standard I/O
				logs.Info("Use FCGI via standard I/O")
			} else {
				logs.Critical("Cannot use FCGI via standard I/O", err)
			}
			return
		}
		if BConfig.Listen.HTTPPort == 0 {
			// remove the Socket file before start
			if utils.FileExists(addr) {
				os.Remove(addr)
			}
			l, err = net.Listen("unix", addr)
		} else {
			l, err = net.Listen("tcp", addr)
		}
		if err != nil {
			logs.Critical("Listen: ", err)
		}
		if err = fcgi.Serve(l, app.Handlers); err != nil {
			logs.Critical("fcgi.Serve: ", err)
		}
		return
	}

	app.Server.Handler = app.Handlers
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] == nil {
			continue
		}
		app.Server.Handler = mws[i](app.Server.Handler)
	}
	app.Server.ReadTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.WriteTimeout = time.Duration(BConfig.Listen.ServerTimeOut) * time.Second
	app.Server.ErrorLog = logs.GetLogger("HTTP")

	// run graceful mode
	if BConfig.Listen.Graceful {
		httpsAddr := BConfig.Listen.HTTPSAddr
		app.Server.Addr = httpsAddr
		if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
			go func() {
				time.Sleep(1000 * time.Microsecond)
				if BConfig.Listen.HTTPSPort != 0 {
					httpsAddr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
					app.Server.Addr = httpsAddr
				}
				server := grace.NewServer(httpsAddr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.EnableMutualHTTPS {
					if err := server.ListenAndServeMutualTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile, BConfig.Listen.TrustCaFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				} else {
					if BConfig.Listen.AutoTLS {
						m := autocert.Manager{
							Prompt:     autocert.AcceptTOS,
							HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
							Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
						}
						app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
						BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
					}
					if err := server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
						logs.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}
			}()
		}
		if BConfig.Listen.EnableHTTP {
			go func() {
				server := grace.NewServer(addr, app.Handlers)
				server.Server.ReadTimeout = app.Server.ReadTimeout
				server.Server.WriteTimeout = app.Server.WriteTimeout
				if BConfig.Listen.ListenTCP4 {
					server.Network = "tcp4"
				}
				if err := server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err, fmt.Sprintf("%d", os.Getpid()))
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}()
		}
		<-endRunning
		return
	}

	// run normal mode
	if BConfig.Listen.EnableHTTPS || BConfig.Listen.EnableMutualHTTPS {
		go func() {
			time.Sleep(1000 * time.Microsecond)
			if BConfig.Listen.HTTPSPort != 0 {
				app.Server.Addr = fmt.Sprintf("%s:%d", BConfig.Listen.HTTPSAddr, BConfig.Listen.HTTPSPort)
			} else if BConfig.Listen.EnableHTTP {
				BeeLogger.Info("Start https server error, conflict with http. Please reset https port")
				return
			}
			logs.Info("https server Running on https://%s", app.Server.Addr)
			if BConfig.Listen.AutoTLS {
				m := autocert.Manager{
					Prompt:     autocert.AcceptTOS,
					HostPolicy: autocert.HostWhitelist(BConfig.Listen.Domains...),
					Cache:      autocert.DirCache(BConfig.Listen.TLSCacheDir),
				}
				app.Server.TLSConfig = &tls.Config{GetCertificate: m.GetCertificate}
				BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile = "", ""
			} else if BConfig.Listen.EnableMutualHTTPS {
				pool := x509.NewCertPool()
				data, err := ioutil.ReadFile(BConfig.Listen.TrustCaFile)
				if err != nil {
					BeeLogger.Info("MutualHTTPS should provide TrustCaFile")
					return
				}
				pool.AppendCertsFromPEM(data)
				app.Server.TLSConfig = &tls.Config{
					ClientCAs:  pool,
					ClientAuth: tls.RequireAndVerifyClientCert,
				}
			}
			if err := app.Server.ListenAndServeTLS(BConfig.Listen.HTTPSCertFile, BConfig.Listen.HTTPSKeyFile); err != nil {
				logs.Critical("ListenAndServeTLS: ", err)
				time.Sleep(100 * time.Microsecond)
				endRunning <- true
			}
		}()

	}
	if BConfig.Listen.EnableHTTP {
		go func() {
			app.Server.Addr = addr
			logs.Info("http server Running on http://%s", app.Server.Addr)
			if BConfig.Listen.ListenTCP4 {
				ln, err := net.Listen("tcp4", app.Server.Addr)
				if err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					return
				}
				if err = app.Server.Serve(ln); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
					return
				}
			} else {
				if err := app.Server.ListenAndServe(); err != nil {
					logs.Critical("ListenAndServe: ", err)
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}
		}()
	}
	<-endRunning
}
```
