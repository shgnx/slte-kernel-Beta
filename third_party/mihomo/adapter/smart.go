package adapter

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/metacubex/mihomo/component/ca"
	"github.com/metacubex/mihomo/component/tls"
	C "github.com/metacubex/mihomo/constant"

	"github.com/metacubex/http"
)

// StatusTest 以浏览器指纹发起一次真实 HTTP 请求，返回响应状态码。
//
// 与 URLTest 的区别：URLTest 只判断"通不通/延迟多少"，StatusTest 判断"目标站是否可用"
// （例如被 403/503 拦截、Cloudflare 拦截等），smart 组据此剔除"延迟低但实际打不开"的节点——
// 正是本项目引入 smart 内核要解决的核心问题。
// 来自上游 mihomo，按原样保留。
func (p *Proxy) StatusTest(ctx context.Context, rawURL string) (status uint16, ok bool, err error) {
	if _, err = urlToMetadata(rawURL); err != nil {
		return 1, false, err
	}

	tlsConfig, err := ca.GetTLSConfig(ca.Option{})
	if err != nil {
		return 1, false, err
	}

	fingerprint, ok2 := tls.GetFingerprint("chrome")
	if !ok2 {
		return 1, false, fmt.Errorf("failed to get TLS fingerprint")
	}

	dialProxy := func(dialCtx context.Context, targetAddr string) (net.Conn, error) {
		var metadata C.Metadata
		if err := metadata.SetRemoteAddress(targetAddr); err != nil {
			return nil, err
		}
		return p.DialContext(dialCtx, &metadata)
	}

	transport := &http.Transport{
		DialContext: func(dialCtx context.Context, network, targetAddr string) (net.Conn, error) {
			return dialProxy(dialCtx, targetAddr)
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ForceAttemptHTTP2:     false,
		DialTLSContext: func(dialCtx context.Context, network, targetAddr string) (net.Conn, error) {
			rawConn, err := dialProxy(dialCtx, targetAddr)
			if err != nil {
				return nil, err
			}
			host, _, splitErr := net.SplitHostPort(targetAddr)
			if splitErr != nil {
				_ = rawConn.Close()
				return nil, splitErr
			}
			uCfg := tls.UConfig(tlsConfig)
			uCfg.ServerName = host
			uConn := tls.UClient(rawConn, uCfg, fingerprint)
			if err := tls.BuildWebsocketHandshakeState(uConn); err != nil {
				_ = rawConn.Close()
				return nil, err
			}
			if err := uConn.HandshakeContext(dialCtx); err != nil {
				_ = rawConn.Close()
				return nil, err
			}
			return uConn, nil
		},
	}

	client := http.Client{
		Timeout:   10 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 3 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	defer client.CloseIdleConnections()

	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return 1, false, err
	}
	req = req.WithContext(ctx)

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/132.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Ch-Ua", `"Not/A)Brand";v="8", "Chromium";v="132", "Google Chrome";v="132"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Sec-Fetch-Dest", "document")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Site", "none")
	req.Header.Set("Sec-Fetch-User", "?1")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	banStatus := map[int]bool{
		http.StatusForbidden:          true, // 403
		http.StatusMethodNotAllowed:   true, // 405
		http.StatusMisdirectedRequest: true, // 421
		http.StatusNotImplemented:     true, // 501
		http.StatusServiceUnavailable: true, // 503
		520:                           true, // Cloudflare 520
		599:                           true, // timeout
	}

	resp, err := client.Do(req)
	var statusCode int
	if err != nil {
		if netErr, okNet := err.(net.Error); okNet && netErr.Timeout() {
			statusCode = 599
		} else if err == context.Canceled || err == context.DeadlineExceeded {
			statusCode = 599
		} else {
			return 1, false, err
		}
	} else {
		statusCode = resp.StatusCode
		ok = !banStatus[statusCode]
		if !ok {
			if statusCode == http.StatusForbidden {
				if resp.Header.Get("Server") == "cloudflare" {
					ok = true
				}
			}
			if statusCode == 520 {
				if resp.Header.Get("Server") != "cloudflare" {
					ok = true
				}
			}
		}
		_ = resp.Body.Close()
	}

	return uint16(statusCode), ok, nil
}
