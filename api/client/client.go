package client

// import (
// 	"crypto/tls"
// 	"fmt"
// 	"net"
// 	"net/http"
// 	"net/url"
// 	"sync"
// 	"time"
// )

// var (
// 	httpCache = make(map[string]*http.Client)
// 	cacheLock sync.Mutex
// )

// // Client is an HTTP REST wrapper. Use one of Get/Post/Put/Delete to get a request
// // object.
// type Client struct {
// 	base        *url.URL
// 	version     string
// 	httpClient  *http.Client
// 	authstring  string
// 	accesstoken string
// 	userAgent   string
// }

// // GetUnixServerPath returns a unix domain socket prepended with the
// // provided path.
// func GetUnixServerPath(socketName string, paths ...string) string {
// 	serverPath := "unix://"
// 	for _, path := range paths {
// 		serverPath = serverPath + path
// 	}
// 	serverPath = serverPath + socketName + ".sock"
// 	return serverPath
// }

// // NewClient returns a new REST client for specified server.
// func NewClient(host, version, userAgent string) (*Client, error) {
// 	baseURL, err := url.Parse(host)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if baseURL.Path == "" {
// 		baseURL.Path = "/"
// 	}
// 	unix2HTTP(baseURL)
// 	hClient := getHTTPClient(host)
// 	if hClient == nil {
// 		return nil, fmt.Errorf("unable to parse provided url: %v", host)
// 	}
// 	c := &Client{
// 		base:        baseURL,
// 		version:     version,
// 		httpClient:  hClient,
// 		authstring:  "",
// 		accesstoken: "",
// 		userAgent:   fmt.Sprintf("%v/%v", userAgent, version),
// 	}
// 	return c, nil
// }

// func unix2HTTP(u *url.URL) {
// 	if u.Scheme == "unix" {
// 		// Override the main URL object so the HTTP lib won't complain
// 		u.Scheme = "http"
// 		u.Host = "unix.sock"
// 		u.Path = ""
// 	}
// }

// func getHTTPClient(host string) *http.Client {
// 	cacheLock.Lock()
// 	defer cacheLock.Unlock()
// 	c, ok := httpCache[host]
// 	if !ok {
// 		u, err := url.Parse(host)
// 		if err != nil {
// 			return nil
// 		}
// 		if u.Path == "" {
// 			u.Path = "/"
// 		}
// 		c = newHTTPClient(u, nil, 10*time.Second, 5*time.Minute)
// 		httpCache[host] = c
// 	}
// 	return c
// }

// func newHTTPClient(
// 	u *url.URL,
// 	tlsConfig *tls.Config,
// 	timeout time.Duration,
// 	responseTimeout time.Duration,
// ) *http.Client {
// 	httpTransport := &http.Transport{
// 		TLSClientConfig: tlsConfig,
// 	}

// 	switch u.Scheme {
// 	case "unix":
// 		socketPath := u.Path
// 		unixDial := func(proto, addr string) (net.Conn, error) {
// 			ret, err := net.DialTimeout("unix", socketPath, timeout)
// 			return ret, err
// 		}
// 		httpTransport.Dial = unixDial
// 		unix2HTTP(u)
// 	default:
// 		httpTransport.Dial = func(proto, addr string) (net.Conn, error) {
// 			return net.DialTimeout(proto, addr, timeout)
// 		}
// 	}

// 	return &http.Client{Transport: httpTransport, Timeout: responseTimeout}
// }
