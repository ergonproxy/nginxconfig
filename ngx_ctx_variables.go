package main

import (
	"context"
	"net/http"
	"os"
	"sync"
	"time"
)

type variables struct{}

// variables that may be available in nginx config for substitution
const (
	vAncientBrowser          = "$ancient_browser"
	vArg                     = "$arg"
	vArgs                    = "$args"
	vBinaryRemoteAddress     = "$binary_remote_addr "
	vBodyBytesSent           = "$body_bytes_sent"
	vBytesReceived           = "$bytes_received"
	vBytesSent               = "$bytes_sent"
	vConnection              = "$connection "
	vConnectionRequests      = "$connection_requests"
	vConnectionActive        = "$connections_active"
	vConnectionReading       = "$connections_reading"
	vConnectionWaiting       = "$connections_waiting"
	vConnectionWriting       = "$connections_writing"
	vContentLength           = "$content_length"
	vContentType             = "$content_type"
	vCookie                  = "$cookie"
	vDateGMT                 = "$date_gmt"
	vDateLocal               = "$date_local"
	vDocumentRoot            = "$document_root"
	vDocumentURI             = "$document_uri"
	vFastCGIPathInfo         = "$fastcgi_path_info"
	vGeoIPAreaCode           = "$geoip_area_code "
	vGeoIPCity               = "$geoip_city  "
	vGeoIPCityContinentCode  = "$geoip_city_continent_code"
	vGeoIPCityCountryCode    = "$geoip_city_country_code"
	vGeoIPCityCountryCode3   = "$geoip_city_country_code3"
	vGeoIPCityCountryName    = "$geoip_city_country_name"
	vGeoIPCountryCode        = "$geoip_country_code"
	vGeoIPCountryCode3       = "$geoip_country_code3"
	vGeoIPCountryName        = "$geoip_country_name"
	vGeoIPDMACode            = "$geoip_dma_name"
	vGeoIPLatitude           = "$geoip_latitude"
	vGeoIPLongitude          = "$geoip_longitude"
	vGeoIPOrg                = "$geoip_org"
	vGeoIPPostalCode         = "$geoip_postal_code"
	vGeoIPRegion             = "$geoip_region"
	vGeoIPRegionName         = "$geoip_region_name"
	vGzipRatio               = "$gzip_ratio"
	vHost                    = "$host"
	vHostname                = "$hostname"
	vHTTP2                   = "$http2"
	vHTTP                    = "$http"
	vHTTPS                   = "$https"
	vInvalidReferer          = "$invalid_referer"
	vIsArgs                  = "$is_args"
	vJWTClaim                = "$jwt_claim"
	vJWTHeader               = "$jwt_header"
	vLimitConnStatus         = "$limit_conn_status"
	vLimitRate               = "$limit_rate"
	vLimitReqStatus          = "$limit_req_status"
	vMemcachedKey            = "$memcached_key"
	vModernBrowser           = "$modern_browser"
	vMsec                    = "$msec"
	vMsie                    = "$msie"
	vNginxVersion            = "$nginx_version"
	vPid                     = "$pid"
	vPipe                    = "$pipe"
	vProtocol                = "$protocol"
	vProxyAddXForwardFor     = "$proxy_add_x_forwarded_for"
	vProxyHost               = "$proxy_host"
	vProxyPort               = "$proxy_port"
	vProxyProtocolAddr       = "$proxy_protocol_addr"
	vProxyProtocolPort       = "$proxy_protocol_port"
	vProxyProtocolServerAddr = "$proxy_protocol_server_addr"
	vProxyProtocolServerPort = "$proxy_protocol_server_port"
	vQueryString             = "$query_string"
	vRealIPRemoteAddr        = "$realip_remote_addr "
	vRealIPRemotePort        = "$realip_remote_port "
	vRealPathRoot            = "$realpath_root "
	vRemoteAddr              = "$remote_addr"
	vRemotePort              = "$remote_port"
	vRemoteUser              = "$remote_user"
	vRequest                 = "$request"
	vRequestBody             = "$request_body"
	vRequestBodyFile         = "$request_body_file"
	vRequestCompletion       = "$request_completion"
	vRequestFilename         = "$request_filename"
	vRequestID               = "$request_id"
	vRequestLength           = "$request_length"
	vRequestMethod           = "$request_method"
	vRequestTime             = "$request_time"
	vRequestURI              = "$request_uri"
	vScheme                  = "$scheme"
	vSecureLink              = "$secure_link"
	vSecureLinkExpires       = "$secure_link_expires"
	vSentHTTP                = "$sent_http"
	vSentTrailer             = "$sent_trailer"
	vServerAddr              = "$server_addr"
	vServerPort              = "$server_port"
	vServerProtocol          = "$server_protocol"
	vSessionLogBinaryID      = "$session_log_binary_id"
	vSessionLogID            = "$session_log_id"
	vSessionTime             = "$session_time"
	vSliceRange              = "$slice_range"
	vSPDY                    = "$spdy"
	vSPDYRequestPriority     = "$spdy_request_priority"
	vSSLCipher               = "$ssl_cipher"
	vSSLCiphers              = "$ssl_ciphers"
	vSSLClientFingerprint    = "$ssl_client_fingerprint"
	vSSLClientIDN            = "$ssl_client_i_dn"
	vSSLClientIDNLegacy      = "$ssl_client_i_dn_legacy"
	vSSLClientRawCert        = "$ssl_client_raw_cert"
	vSSLClientSDN            = "$ssl_client_s_dn"
	vSSLClientSDNLegacy      = "$ssl_client_s_dn_legacy"
	vSSLClientSerial         = "$ssl_client_serial"
	vSSLClientVEnd           = "$ssl_client_v_end"
	vSSLClientVReamin        = "$ssl_client_v_remain"
	vSSLClientVStart         = "$ssl_client_v_start"
	vSSLClientVerify         = "$ssl_client_verify"
	vSSLCurves               = "$ssl_curves"
	vSSLEarlyData            = "$ssl_early_data"
	vSSLPrereadAlpnProtocols = "$ssl_preread_alpn_protocols"
	vSSLPrereadProtocols     = "$ssl_preread_protocols"
	vSSLPrereadServerName    = "$ssl_preread_server_name"
	vSSLProtocol             = "$ssl_protocol"
	vSSLServerName           = "$ssl_server_name"
	vSSLSessionID            = "$ssl_session_id"
	vSSLSessionReused        = "$ssl_session_reused"
	vStatus                  = "$status"
	vTCPInfoRtt              = "$tcpinfo_rtt"
	vTCPInfoRttVar           = "$tcpinfo_rttvar"
	vTCPInfoSndCwnd          = "$tcpinfo_snd_cwnd"
	vTCPInfoRcvSpace         = "$tcpinfo_rcv_space"
	vTimeISO8601             = "$time_iso8601"
	vTimeLocal               = "$time_local"
	vUIDGot                  = "$uid_got"
	vUIDReset                = "$uid_reset"
	vUIDSet                  = "$uid_set"
	vUpstreamAddr            = "$upstream_addr"
	vUpstreamBytesReceived   = "$upstream_bytes_received"
	vUpstreamBytesSent       = "$upstream_bytes_sent"
	vUpstreamCacheStatu      = "$upstream_cache_status"
	vUpstreamConnectTime     = "$upstream_connect_time"
	vUpstreamCookie          = "$upstream_cookie"
	vUpstreamFirstByteTime   = "$upstream_first_byte_time"
	vUpstreamHeaderTime      = "$upstream_header_time"
	vUpstreamHTTP            = "$upstream_http"
	vUpstreamQueueTime       = "$upstream_queue_time"
	vUpstreamResponseLength  = "$upstream_response_length"
	vUpstreamResponseTime    = "$upstream_response_time"
	vUpstreamSessionTime     = "$upstream_session_time"
	vUpstreamStatus          = "$upstream_status"
	vUpstreamTrailer         = "$upstream_trailer"
	vURI                     = "$uri"
)

// extra ctx keys
type (
	uriKey     struct{}
	tlsModeKey struct{}
	requestID  struct{}
)
type variableFunc func() interface{}

func createVariables() *sync.Map {
	m := new(sync.Map)
	return m
}

func setTimeVariables(m *sync.Map) {
	cache := cachedTimeFunc()
	m.Store(vDateGMT, cache(func(ts time.Time) string {
		return ts.Format(http.TimeFormat)
	}))
	m.Store(vDateLocal, cache(func(ts time.Time) string {
		return ts.Format(time.RFC1123)
	}))
	m.Store(vTimeISO8601, cache(func(ts time.Time) string {
		return ts.Format(iso8601Milli)
	}))
	m.Store(vTimeLocal, cache(func(ts time.Time) string {
		return ts.Local().Format(commonLogFormatTime)
	}))
}

func setVariable(ctx context.Context, key, value interface{}) {
	_, ok := key.(string)
	if !ok {
		return
	}
	if v := ctx.Value(variables{}); v != nil {
		v.(*sync.Map).Store(key, value)
	}
}

func getVariable(m *sync.Map, key interface{}) interface{} {
	_, ok := key.(string)
	if !ok {
		return nil
	}
	if v, ok := m.Load(variables{}); ok {
		switch e := v.(type) {
		case variableFunc:
			return e()
		default:
			return e
		}
	}
	return nil
}

func cachedFunc(f func() interface{}) func() interface{} {
	var n interface{}
	return func() interface{} {
		if n != nil {
			return n
		}
		if f != nil {
			n = f()
		}
		return n
	}
}

func cachedTimeFunc() func(func(time.Time) string) variableFunc {
	cache := cachedFunc(func() interface{} {
		return time.Now()
	})
	return func(f func(time.Time) string) variableFunc {
		return func() interface{} {
			now := cache().(time.Time)
			return f(now)
		}
	}
}

func setRequestVariables(m *sync.Map, r *http.Request) {
	// seeting query variables
	query := r.URL.Query()
	for k := range query {
		// setting $arg_{query_name}
		m.Store(vArg+"_"+k, query.Get(k))
	}
	m.Store(vArgs, r.URL.RawQuery)
	m.Store(vContentLength, r.Header.Get("Content-Length"))
	m.Store(vContentType, r.Header.Get("Content-Type"))
	for _, cookie := range r.Cookies() {
		m.Store(vCookie+"_"+cookie.Name, cookie.Value)
	}
	ctx := r.Context()
	root := ""
	if v := ctx.Value(rootKey{}); v != nil {
		root = v.(string)
	} else if v := ctx.Value(aliasKey{}); v != nil {
		root = v.(string)
	}
	m.Store(vDocumentRoot, root)
	m.Store(vDocumentURI, ctx.Value(uriKey{}))
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	if host == "" {
		if v := ctx.Value(serverNameKey{}); v != nil {
			host = v.(string)
		}
	}
	m.Store(vHost, host)
	n, _ := os.Hostname() //TODO: handle errors
	m.Store(vHostname, n)
	for h := range r.Header {
		m.Store(vHTTP+"_"+h, r.Header.Get(h))
	}
	a := ""
	if len(query) > 0 {
		a = "?"
	}
	m.Store(vIsArgs, a)
}
