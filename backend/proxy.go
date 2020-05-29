package backend

import (
	"net/http"
	"net/http/httputil"
	"strings"

	"github.com/sirupsen/logrus"
)

var (
	_ http.Handler = &Proxy{}
)

type Proxy struct {
	OriginAddress  string      `hcl:"origin_address"`
	OriginHost     string      `hcl:"origin_host"`
	OriginScheme   string      `hcl:"origin_scheme,optional"` // optional defaults to attr
	RequestHeaders http.Header `hcl:"request_headers"`
	rp             *httputil.ReverseProxy
	log            *logrus.Entry
}

func NewProxy() func(*logrus.Entry) http.Handler {
	return func(log *logrus.Entry) http.Handler {
		proxy := &Proxy{log: log}
		proxy.rp = &httputil.ReverseProxy{Director: proxy.director}
		return proxy
	}
}

func (p *Proxy) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	p.rp.ServeHTTP(rw, req)
}

// director request modification before roundtrip
func (p *Proxy) director(req *http.Request) {
	req.URL.Host = p.OriginAddress
	req.URL.Scheme = "http"
	if strings.HasSuffix(p.OriginAddress, "443") {
		req.URL.Scheme = "https" // TODO: improve conf options, scheme or url
	}
	req.Host = p.OriginHost

	for header, value := range p.RequestHeaders {
		req.Header.Set(header, value[0])
	}
	p.log.WithField("uid", req.Context().Value("requestID")).WithField("custom-header", p.RequestHeaders).Debug()
}

func (p *Proxy) String() string {
	return "Proxy"
}
