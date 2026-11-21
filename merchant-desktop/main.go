// 商家端桌面版：内嵌前端静态资源，本地启动 HTTP 服务。
// 以系统托盘图标常驻（无黑窗口），双击/菜单可打开浏览器。
package main

import (
	"embed"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/energye/systray"
)

//go:embed all:web
var webFS embed.FS

//go:embed icon.ico
var iconBytes []byte

// 固定端口：保证 localStorage（登录 token）在同一 origin 下持续有效。
const addr = "127.0.0.1:8788"

func main() {
	sub, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("嵌入资源失败: %v", err)
	}
	url := "http://" + addr + "/"

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		// 端口占用：多半已有一个实例在运行，直接打开浏览器后退出。
		openBrowser(url)
		return
	}

	go func() {
		if err := http.Serve(ln, newHandler(sub)); err != nil {
			log.Printf("http serve: %v", err)
		}
	}()

	// 调试/测试：NO_TRAY=1 时不显示托盘，仅后台服务。
	if os.Getenv("NO_TRAY") != "" {
		select {}
	}

	systray.Run(func() {
		systray.SetIcon(iconBytes)
		systray.SetTitle("阡月里商家中心")
		systray.SetTooltip("阡月里商家中心")
		systray.CreateMenu()

		mOpen := systray.AddMenuItem("打开商家中心", "在浏览器中打开")
		mOpen.Click(func() { openBrowser(url) })

		systray.AddSeparator()
		mQuit := systray.AddMenuItem("退出", "关闭服务并退出")
		mQuit.Click(func() { systray.Quit() })

		systray.SetOnClick(func(m systray.IMenu) {
			if m != nil {
				_ = m.ShowMenu()
			}
		})
		systray.SetOnDClick(func(m systray.IMenu) { openBrowser(url) })

		if os.Getenv("NO_OPEN") == "" {
			openBrowser(url)
		}
	}, func() {})
}

func openBrowser(url string) {
	_ = exec.Command("cmd", "/c", "start", "", url).Start()
}

// newHandler：/api 反向代理到线上 API，其余走内嵌静态资源。
// 这样桌面版同源请求，不依赖后端 CORS 白名单。
func newHandler(root fs.FS) http.Handler {
	static := spaHandler(root)
	target, err := url.Parse("http://localhost:8080")
	if err != nil {
		log.Fatalf("API 地址解析失败: %v", err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	director := proxy.Director
	proxy.Director = func(req *http.Request) {
		director(req)
		req.Host = target.Host
	}
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("API 代理失败: %v", err)
		http.Error(w, "API 暂不可用", http.StatusBadGateway)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			proxy.ServeHTTP(w, r)
			return
		}
		static.ServeHTTP(w, r)
	})
}

// spaHandler 静态资源服务：前端使用 history 路由。
// 未命中的页面路径回退到 index.html（静态资源仍返回 404）。
func spaHandler(root fs.FS) http.Handler {
	fileServer := http.FileServer(http.FS(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(root, name); err != nil && path.Ext(name) == "" {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	})
}
