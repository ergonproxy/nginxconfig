package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.cloudfoundry.org/bytefmt"
)

type (
	defaultPortKey struct{}
)

var tlsCache = new(sync.Map)

type listenOpts struct {
	net           string
	addrPort      string
	defaultServer bool
	ssl           bool
	http2         bool
	spdy          bool
	proxyProtocol bool

	//ssl related options
	sslOpts sslOptions
	manager *connManager
}

var _ tls.ClientSessionCache = tlsClientCache{}

type tlsClientCache struct{}

func (tlsClientCache) Get(key string) (*tls.ClientSessionState, bool) {
	v, ok := tlsCache.Load(key)
	if !ok {
		return nil, ok
	}
	return v.(*tls.ClientSessionState), ok
}

func (tlsClientCache) Put(key string, cs *tls.ClientSessionState) {
	tlsCache.Store(key, cs)
}

type sslOptions struct {
	bufferSize          intValue
	certificate         stringValue
	certificateKey      stringValue
	clientCertificate   stringValue
	ciphers             stringSliceValue
	crl                 stringValue
	dhParam             stringValue
	earlData            boolValue
	ecdheCUrve          stringValue
	passwordFile        stringValue
	preferServerCiphers boolValue
	protocols           stringSliceValue
	sessionCache        stringValue
	sessionTicketKey    stringValue
	sessionTickets      boolValue
	timeout             durationValue
	stapling            boolValue
	staplingFile        stringValue
	staplingResponder   stringValue
	staplingVerify      boolValue
	trustedCertificate  stringValue
}

func (ss sslOptions) config() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(ss.certificate.value, ss.certificateKey.value)
	if err != nil {
		return nil, err
	}
	c := &tls.Config{Certificates: []tls.Certificate{cert}}
	if ss.ciphers.set {
		for _, f := range ss.ciphers.value {
			c.CipherSuites = append(c.CipherSuites, standardCipherSuits[f])
		}
	}
	if ss.preferServerCiphers.set {
		c.PreferServerCipherSuites = ss.preferServerCiphers.value
	}
	if ss.sessionTickets.set {
		c.SessionTicketsDisabled = !ss.sessionTickets.value
	}
	if ss.sessionTicketKey.set {
		b, err := ioutil.ReadFile(ss.sessionTicketKey.value)
		if err != nil {
			return nil, err
		}
		var k [32]byte
		copy(k[:], b)
		c.SetSessionTicketKeys([][32]byte{k})
	}
	if ss.sessionCache.set {
		c.ClientSessionCache = tlsClientCache{}
	}
	if ss.protocols.set {
		c.NextProtos = append(c.NextProtos, ss.protocols.value...)
	}
	return c, nil
}

func (ss *sslOptions) init() {
	ss.earlData.store(false)
	ss.ecdheCUrve.store("auto")
	ss.preferServerCiphers.store(false)
	ss.sessionCache.store("none")
	ss.protocols.store([]string{"TLSv1", "TLSv1.1", "TLSv1.2"})
	ss.sessionTickets.store(true)
	ss.timeout.store(5 * time.Minute)
	ss.stapling.store(false)
	ss.staplingVerify.store(false)
}

func (ss *sslOptions) load(r *rule) error {
	switch r.name {
	case "ssl_buffer_size":
		if len(r.args) > 0 {
			v, err := bytefmt.ToBytes(r.args[0])
			if err != nil {
				return err
			}
			ss.bufferSize.store(int64(v))
		}
	case "ssl_certificate":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.certificate.store(file)
		}
	case "ssl_certificate_key":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.certificateKey.store(file)
		}
	case "ssl_ciphers":
		if len(r.args) > 0 {
			c, err := openSSLCiphers(r.args[0])
			if err != nil {
				return err
			}
			var p []string
			for _, v := range standardCiphers(c) {
				p = append(p, v.String())
			}
			ss.ciphers.store(p)
		}
	case "ssl_client_certificate":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.clientCertificate.store(file)
		}
	case "ssl_crl":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.crl.store(file)
		}
	case "ssl_dhparam":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.dhParam.store(file)
		}
	case "ssl_early_data":
		if len(r.args) > 0 {
			switch r.args[0] {
			case "on":
				ss.earlData.store(true)
			case "off":
				ss.earlData.store(false)
			}
		}
	case "ssl_ecdh_curve":
		if len(r.args) > 0 {
			ss.ecdheCUrve.store(r.args[0])
		}
	case "ssl_password_file":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.dhParam.store(file)
		}
	case "ssl_prefer_server_ciphers":
		if len(r.args) > 0 {
			switch r.args[0] {
			case "on":
				ss.preferServerCiphers.store(true)
			case "off":
				ss.preferServerCiphers.store(false)
			}
		}
	case "ssl_protocols":
		if len(r.args) > 0 {
			ss.protocols.store(r.args)
		}
	case "ssl_session_cache":
		if len(r.args) > 0 {
			ss.ecdheCUrve.store(r.args[0])
		}
	case "ssl_session_ticket_key":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.sessionTicketKey.store(file)
		}
	case "ssl_session_tickets":
		if len(r.args) > 0 {
			switch r.args[0] {
			case "on":
				ss.sessionTickets.store(true)
			case "off":
				ss.sessionTickets.store(false)
			}
		}
	case "ssl_session_timeout":
		if len(r.args) > 0 {
			d, err := time.ParseDuration(r.args[0])
			if err != nil {
				return err
			}
			ss.timeout.store(d)
		}
	case "ssl_stapling":
		if len(r.args) > 0 {
			switch r.args[0] {
			case "on":
				ss.stapling.store(true)
			case "off":
				ss.stapling.store(false)
			}
		}
	case "ssl_stapling_file":
		if len(r.args) > 0 {
			file := r.args[0]
			if err := checkFile(file); err != nil {
				return err
			}
			ss.staplingFile.store(file)
		}
	case "ssl_stapling_responder":
		if len(r.args) > 0 {
			u := r.args[0]
			if _, err := url.Parse(u); err != nil {
				return err
			}
			ss.staplingResponder.store(u)
		}
	case "ssl_stapling_verify":
		if len(r.args) > 0 {
			switch r.args[0] {
			case "on":
				ss.staplingVerify.store(true)
			case "off":
				ss.staplingVerify.store(false)
			}
		}
	case "ssl_trusted_certificate":
	}

	return nil
}

type cipher struct {
	proto   string
	kx      string // key exchange
	auth    string
	enc     string
	encSize uint16
	encPad  string
	mac     string
}

func (c *cipher) init() {
	c.proto = "TLS"
	c.kx = "RSA"
	c.auth = "RSA"
	c.enc = "AES"
	c.encSize = 128
	c.encPad = "CBC"
	c.mac = "SHA"
}

func (c cipher) String() string {
	var s string
	if c.kx == c.auth {
		s = fmt.Sprintf("%s_%s_WITH_%s", c.proto, c.kx, c.enc)
	} else {
		s = fmt.Sprintf("%s_%s_%s_WITH_%s", c.proto, c.kx, c.auth, c.enc)
	}
	if c.enc != "3DES_EDE" && c.enc != "SEED" {
		s += fmt.Sprintf("_%d", c.encSize)
	}
	if c.enc != "RC4" {
		s += "_" + c.encPad
	}
	return s + "_" + c.mac
}

var kxProto = map[string]string{
	"AECDH": "ECDH_anon",
	"ADH":   "DH_anon",
	"DH":    "DH",
	"DHE":   "DHE",
	"ECDH":  "ECDH",
	"ECDHE": "ECDHE",
	"EDH":   "EDH",
	"PSK":   "PSK",
	"RSA":   "RSA",
	"SRP":   "SRP",
}
var authProto = map[string]bool{
	"DSS":   true,
	"ECDSA": true,
	"PSK":   true,
	"RSA":   true,
	"SRP":   true,
}

var digests = map[string]bool{
	"MD5":    true,
	"SHA":    true,
	"SHA256": true,
	"SHA384": true,
}

var ciphers = map[string]struct {
	enc  string
	size uint16
}{
	"AES":         {"AES", 128},
	"AES128":      {"AES", 128},
	"AES256":      {"AES", 126},
	"3DES":        {"3DES_EDE", 112},
	"DES":         {"3DES_EDE", 112},
	"RC4":         {"RC4", 128},
	"SEED":        {"SEED", 128},
	"CAMELLIA128": {"CAMELLIA128", 128},
	"CAMELLIA256": {"CAMELLIA128", 256},
	"IDEA":        {"IDEA", 128},
}

var standardCipherSuits = map[string]uint16{
	"TLS_RSA_WITH_RC4_128_SHA":                 tls.TLS_RSA_WITH_RC4_128_SHA,
	"TLS_RSA_WITH_3DES_EDE_CBC_SHA":            tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA ":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
	"TLS_RSA_WITH_AES_256_CBC_SHA ":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
	"TLS_RSA_WITH_AES_128_CBC_SHA256":          tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_RSA_WITH_AES_128_GCM_SHA256":          tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_RSA_WITH_AES_256_GCM_SHA384":          tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":         tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":     tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":     tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_RC4_128_SHA ":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
	"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA ":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
	"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA ":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
	"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256 ": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":    tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":    tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 ": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":    tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 ": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":     tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":   tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,

	// TLS 1.3 cipher suites.
	"TLS_AES_128_GCM_SHA256  ":      tls.TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384  ":      tls.TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256 ": tls.TLS_CHACHA20_POLY1305_SHA256,

	// TLS_FALLBACK_SCSV isn't a standard cipher suite but an indicator
	// that the client is doing version fallback. See RFC 7507.
	"TLS_FALLBACK_SCSV ": tls.TLS_FALLBACK_SCSV,
}

func (c *cipher) parse(spec string) {
	c.init()
	parts := strings.Split(spec, "-")
	idx := 0
	if kx, ok := kxProto[parts[0]]; ok {
		c.kx = kx
		if authProto[parts[1]] {
			c.auth = parts[1]
			idx = 2
		} else {
			c.auth = c.kx
			idx = 1
		}
	}
	if v, ok := ciphers[parts[idx]]; ok {
		c.enc = v.enc
		c.encSize = v.size
	}
	if len(parts) > idx+1 && digests[parts[idx+1]] {
		c.mac = parts[idx+1]
	} else {
		if len(parts) > idx+1 {
			c.encPad = parts[idx+1][0:3]
		}
		if len(parts) > idx+2 {
			c.mac = parts[idx+2]
		}
	}
}

// requires openssl present
func openSSLCiphers(a string) ([]cipher, error) {
	out, err := exec.Command("openssl", "ciphers", a).CombinedOutput()
	if err != nil {
		return nil, err
	}
	parts := strings.Split(string(bytes.TrimSpace(out)), ":")
	var result []cipher
	for _, v := range parts {
		c := cipher{}
		c.parse(v)
		result = append(result, c)
	}
	return result, nil
}

func standardCiphers(c []cipher) []cipher {
	var o []cipher
	for _, v := range c {
		if _, ok := standardCipherSuits[v.String()]; ok {
			o = append(o, v)
		}
	}
	return o
}

func checkFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}
	return f.Close()
}

type stringValue struct {
	set   bool
	value string
}

func (s *stringValue) store(v string) {
	s.value = v
	s.set = true
}

func (s stringValue) merge(other stringValue) stringValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type intValue struct {
	set   bool
	value int64
}

func (s *intValue) store(v int64) {
	s.value = v
	s.set = true
}

func (s intValue) merge(other intValue) intValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type boolValue struct {
	set   bool
	value bool
}

func (s *boolValue) store(v bool) {
	s.value = v
	s.set = true
}

func (s boolValue) merge(other boolValue) boolValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type stringSliceValue struct {
	set   bool
	value []string
}

func (s *stringSliceValue) store(v []string) {
	s.value = v
	s.set = true
}

func (s stringSliceValue) merge(other stringSliceValue) stringSliceValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type durationValue struct {
	set   bool
	value time.Duration
}

func (s *durationValue) store(v time.Duration) {
	s.value = v
	s.set = true
}

func (s durationValue) merge(other durationValue) durationValue {
	if other.set {
		s.value = other.value
	}
	return s
}

type interfaceValue struct {
	set   bool
	value interface{}
}

func (s *interfaceValue) store(v interfaceValue) {
	s.value = v
	s.set = true
}

func (s interfaceValue) merge(other interfaceValue) interfaceValue {
	if other.set {
		s.value = other.value
	}
	return s
}

var ngxPort atomic.Value

func defaultPort() int {
	if v := ngxPort.Load(); v != nil {
		return v.(int)
	}
	// try 80
	p := 80
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", p))
	if err != nil {
		p = 8000
		l, err = net.Listen("tcp", fmt.Sprintf(":%d", p))
	}
	if err != nil {
		panic("vince: failed to bind to default port " + err.Error())
	}
	l.Close()
	ngxPort.Store(p)
	return p
}

func parseListen(r *rule, defaultPort string) listenOpts {
	var ls listenOpts
	if len(r.args) > 0 {
		a := r.args[0]
		if _, err := strconv.Atoi(a); err == nil {
			ls.net = "tcp"
			ls.addrPort = net.IPv4zero.String() + ":" + a
		} else if ip := net.ParseIP(a); ip != nil {
			ls.net = "tcp"
			ls.addrPort = ip.String() + ":" + defaultPort
		} else if h, p, err := net.SplitHostPort(a); err == nil {
			if h == "unix" {
				ls.net = h
				ls.addrPort = p
			} else {
				ls.net = "tcp"
				ls.addrPort = a
			}
		} else {
			switch a {
			case "localhost", "[::]", "[::1]":
				ls.net = "tcp"
				ls.addrPort = a + ":" + defaultPort
			default:
				u, err := url.Parse(a)
				if err == nil {
					ls.net = u.Scheme
					ls.addrPort = u.Host
					//TODO: ensure there is port set
				}
			}
		}
		if len(r.args) > 1 {
			for _, a := range r.args[1:] {
				switch a {
				case "default_server":
					ls.defaultServer = true
				case "ssl":
					ls.ssl = true
				case "http2":
					ls.http2 = true
				case "spdy":
					ls.spdy = true
				case "proxy_protocol":
					ls.proxyProtocol = true
				}
			}
		}
	}
	if ls.ssl {
		// load ssl
		serverRule := r.parent
		httpRule := serverRule.parent
		for _, b := range httpRule.children {
			if err := ls.sslOpts.load(b); err != nil {
				//TODO: return error?
				break
			}
		}
		for _, b := range serverRule.children {
			if err := ls.sslOpts.load(b); err != nil {
				//TODO: return error?
				break
			}
		}
	}
	return ls
}

type httpCoreConfig struct {
	root   stringValue
	alias  stringValue
	client struct {
		body struct {
			bufferSize intValue
			timeout    durationValue
			maxSize    intValue
		}
	}
}

func (c *httpCoreConfig) load(r *rule) error {
	switch r.name {
	case "root":
		c.root.store(r.args[0])
	case "alias":
		c.alias.store(r.args[0])
	case "client_body_buffer_size":
		v, err := bytefmt.ToBytes(r.args[0])
		if err != nil {
			return err
		}
		c.client.body.bufferSize.store(int64(v))
	case "client_body_timeout":
		v, err := time.ParseDuration(r.args[0])
		if err != nil {
			return err
		}
		c.client.body.timeout.store(v)
	case "client_max_body_size":
		v, err := bytefmt.ToBytes(r.args[0])
		if err != nil {
			return err
		}
		c.client.body.bufferSize.store(int64(v))
	}
	return nil
}
