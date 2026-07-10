package netproxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

var (
	forwarderMu sync.Mutex
	forwarders  = map[string]string{} // normalized upstream URL -> local http://127.0.0.1:port
)

// BrowserProxyServer returns --proxy-server value for Chrome.
// When upstream proxy requires auth, a local no-auth forwarder is started automatically.
func BrowserProxyServer(proxyURL string) (string, error) {
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return "", nil
	}
	if !strings.Contains(proxyURL, "://") {
		proxyURL = "http://" + proxyURL
	}

	u, err := url.Parse(proxyURL)
	if err != nil {
		return "", fmt.Errorf("parse proxy URL: %w", err)
	}
	if u.User == nil {
		return ChromeProxyServer(proxyURL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("authenticated browser proxy requires http/https upstream, got %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("proxy URL missing host")
	}

	forwarderMu.Lock()
	defer forwarderMu.Unlock()
	if local, ok := forwarders[proxyURL]; ok {
		return local, nil
	}

	f, err := startHTTPForwarder(u)
	if err != nil {
		return "", err
	}
	forwarders[proxyURL] = f.localURL
	return f.localURL, nil
}

type httpForwarder struct {
	upstream *url.URL
	authHdr  string
	localURL string
}

func startHTTPForwarder(upstream *url.URL) (*httpForwarder, error) {
	authHdr := ""
	if upstream.User != nil {
		pass, _ := upstream.User.Password()
		token := base64.StdEncoding.EncodeToString([]byte(upstream.User.Username() + ":" + pass))
		authHdr = "Basic " + token
	}

	upstreamNoAuth := *upstream
	upstreamNoAuth.User = nil

	f := &httpForwarder{
		upstream: &upstreamNoAuth,
		authHdr:  authHdr,
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen local forwarder: %w", err)
	}

	f.localURL = "http://" + ln.Addr().String()
	srv := &http.Server{Handler: http.HandlerFunc(f.serve)}
	go func() {
		_ = srv.Serve(ln)
	}()
	return f, nil
}

func (f *httpForwarder) serve(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodConnect {
		f.serveConnect(w, r)
		return
	}
	f.serveHTTP(w, r)
}

func (f *httpForwarder) serveConnect(w http.ResponseWriter, r *http.Request) {
	upstreamConn, err := net.Dial("tcp", f.upstream.Host)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer upstreamConn.Close()

	target := r.Host
	if target == "" {
		target = r.URL.Host
	}
	req := fmt.Sprintf("CONNECT %s HTTP/1.1\r\nHost: %s\r\n", target, target)
	if f.authHdr != "" {
		req += "Proxy-Authorization: " + f.authHdr + "\r\n"
	}
	req += "\r\n"
	if _, err := io.WriteString(upstreamConn, req); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	br := bufio.NewReader(upstreamConn)
	resp, err := http.ReadResponse(br, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	if resp.StatusCode != http.StatusOK {
		http.Error(w, resp.Status, resp.StatusCode)
		return
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "hijack not supported", http.StatusInternalServerError)
		return
	}
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer clientConn.Close()

	if br.Buffered() > 0 {
		if _, err := io.Copy(upstreamConn, br); err != nil {
			return
		}
	}

	_, _ = clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))
	go io.Copy(upstreamConn, clientConn)
	io.Copy(clientConn, upstreamConn)
}

func (f *httpForwarder) serveHTTP(w http.ResponseWriter, r *http.Request) {
	transport := &http.Transport{
		Proxy: http.ProxyURL(f.upstream),
	}
	transport.Proxy = func(*http.Request) (*url.URL, error) {
		return f.upstream, nil
	}
	// Inject auth on every proxied hop.
	client := &http.Client{
		Transport: roundTripperWithProxyAuth{base: transport, authHdr: f.authHdr},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	outReq := r.Clone(r.Context())
	outReq.RequestURI = ""
	if f.authHdr != "" {
		outReq.Header.Set("Proxy-Authorization", f.authHdr)
	}
	resp, err := client.Do(outReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, vals := range resp.Header {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

type roundTripperWithProxyAuth struct {
	base    http.RoundTripper
	authHdr string
}

func (rt roundTripperWithProxyAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.authHdr != "" {
		req = req.Clone(req.Context())
		req.Header.Set("Proxy-Authorization", rt.authHdr)
	}
	return rt.base.RoundTrip(req)
}
