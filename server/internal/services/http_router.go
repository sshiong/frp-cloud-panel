package services

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"

	"github.com/frp-cloud-panel/server/internal/database"
	"github.com/frp-cloud-panel/server/internal/models"
)

// HTTPRouter HTTP/HTTPS 路由服务
type HTTPRouter struct {
	certDir    string
	certCache  map[string]*tls.Certificate
	certMutex  sync.RWMutex
	proxyMap   map[string]*httputil.ReverseProxy
	proxyMutex sync.RWMutex
}

// NewHTTPRouter 创建新的 HTTP 路由器
func NewHTTPRouter(certDir string) *HTTPRouter {
	return &HTTPRouter{
		certDir:   certDir,
		certCache: make(map[string]*tls.Certificate),
		proxyMap:  make(map[string]*httputil.ReverseProxy),
	}
}

// Start 启动 HTTP/HTTPS 服务器
func (r *HTTPRouter) Start(httpPort, httpsPort int) error {
	// 启动 HTTP 服务器
	go r.startHTTPServer(httpPort)

	// 启动 HTTPS 服务器
	go r.startHTTPSServer(httpsPort)

	return nil
}

// startHTTPServer 启动 HTTP 服务器
func (r *HTTPRouter) startHTTPServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handleHTTP)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("HTTP server starting on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Printf("HTTP server error: %v", err)
	}
}

// startHTTPSServer 启动 HTTPS 服务器
func (r *HTTPRouter) startHTTPSServer(port int) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", r.handleHTTPS)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("HTTPS server starting on %s", addr)

	// 配置 TLS
	tlsConfig := &tls.Config{
		GetCertificate: r.getCertificate,
	}

	server := &http.Server{
		Addr:      addr,
		Handler:   mux,
		TLSConfig: tlsConfig,
	}

	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Printf("HTTPS server error: %v", err)
	}
}

// handleHTTP 处理 HTTP 请求
func (r *HTTPRouter) handleHTTP(w http.ResponseWriter, req *http.Request) {
	host := req.Host

	// 检查是否需要重定向到 HTTPS
	if r.shouldRedirectToHTTPS(host) {
		httpsURL := fmt.Sprintf("https://%s%s", host, req.URL.String())
		http.Redirect(w, req, httpsURL, http.StatusMovedPermanently)
		return
	}

	// 获取代理
	proxy, err := r.getProxy(host)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	// 代理请求
	proxy.ServeHTTP(w, req)
}

// handleHTTPS 处理 HTTPS 请求
func (r *HTTPRouter) handleHTTPS(w http.ResponseWriter, req *http.Request) {
	host := req.Host

	// 获取代理
	proxy, err := r.getProxy(host)
	if err != nil {
		http.Error(w, "Service not found", http.StatusNotFound)
		return
	}

	// 代理请求
	proxy.ServeHTTP(w, req)
}

// getCertificate 获取 TLS 证书
func (r *HTTPRouter) getCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	domain := hello.ServerName

	// 检查缓存
	r.certMutex.RLock()
	if cert, ok := r.certCache[domain]; ok {
		r.certMutex.RUnlock()
		return cert, nil
	}
	r.certMutex.RUnlock()

	// 从数据库获取域名信息
	var domainInfo models.Domain
	if err := database.DB.Where("domain = ?", domain).First(&domainInfo).Error; err != nil {
		return nil, fmt.Errorf("domain not found: %s", domain)
	}

	// 检查 HTTPS 模式
	if domainInfo.HTTPSMode == "none" {
		return nil, fmt.Errorf("HTTPS not enabled for domain: %s", domain)
	}

	// 加载证书
	cert, err := r.loadCertificate(domain)
	if err != nil {
		return nil, err
	}

	// 缓存证书
	r.certMutex.Lock()
	r.certCache[domain] = cert
	r.certMutex.Unlock()

	return cert, nil
}

// loadCertificate 加载证书
func (r *HTTPRouter) loadCertificate(domain string) (*tls.Certificate, error) {
	// 从证书目录加载
	certPath := fmt.Sprintf("%s/%s.crt", r.certDir, domain)
	keyPath := fmt.Sprintf("%s/%s.key", r.certDir, domain)

	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate for %s: %w", domain, err)
	}

	return &cert, nil
}

// getProxy 获取反向代理
func (r *HTTPRouter) getProxy(host string) (*httputil.ReverseProxy, error) {
	// 检查缓存
	r.proxyMutex.RLock()
	if proxy, ok := r.proxyMap[host]; ok {
		r.proxyMutex.RUnlock()
		return proxy, nil
	}
	r.proxyMutex.RUnlock()

	// 从数据库获取映射信息
	var mapping models.ProxyMapping
	if err := database.DB.Where("domain = ? AND status = ?", host, "running").First(&mapping).Error; err != nil {
		return nil, fmt.Errorf("no mapping found for host: %s", host)
	}

	// 创建目标 URL
	targetURL, err := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", mapping.RemotePort))
	if err != nil {
		return nil, err
	}

	// 创建反向代理
	proxy := httputil.NewSingleHostReverseProxy(targetURL)

	// 自定义错误处理
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("Proxy error for %s: %v", host, err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}

	// 缓存代理
	r.proxyMutex.Lock()
	r.proxyMap[host] = proxy
	r.proxyMutex.Unlock()

	return proxy, nil
}

// shouldRedirectToHTTPS 检查是否需要重定向到 HTTPS
func (r *HTTPRouter) shouldRedirectToHTTPS(host string) bool {
	var domain models.Domain
	if err := database.DB.Where("domain = ?", host).First(&domain).Error; err != nil {
		return false
	}

	return domain.HTTPSMode != "none"
}

// ReloadCertificates 重新加载证书
func (r *HTTPRouter) ReloadCertificates() {
	r.certMutex.Lock()
	r.certCache = make(map[string]*tls.Certificate)
	r.certMutex.Unlock()
	log.Println("Certificate cache cleared")
}

// ClearProxyCache 清除代理缓存
func (r *HTTPRouter) ClearProxyCache() {
	r.proxyMutex.Lock()
	r.proxyMap = make(map[string]*httputil.ReverseProxy)
	r.proxyMutex.Unlock()
	log.Println("Proxy cache cleared")
}

// GetStats 获取路由器统计信息
func (r *HTTPRouter) GetStats() map[string]interface{} {
	r.certMutex.RLock()
	certCount := len(r.certCache)
	r.certMutex.RUnlock()

	r.proxyMutex.RLock()
	proxyCount := len(r.proxyMap)
	r.proxyMutex.RUnlock()

	return map[string]interface{}{
		"cached_certs": certCount,
		"cached_proxies": proxyCount,
		"uptime": time.Now().Format(time.RFC3339),
	}
}
