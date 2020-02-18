package main

import (
	"context"
	"net/http"
	"os"
)

// time formats
const (
	iso8601Milli        = "2006-01-02T15:04:05.000Z"
	commonLogFormatTime = "02/Jan/2006:15:04:05 -0700"
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
const (
	vRequestMatchKind = "request_match_kind"
)

// extra ctx keys
type (
	requestID struct{}
)

func setVariable(ctx context.Context, key string, value interface{}) {
	if v := ctx.Value(variables{}); v != nil {
		v.(map[string]interface{})[key] = value
	}
}

func setRequestVariables(m map[string]interface{}, r *http.Request) {
	// seeting query variables
	query := r.URL.Query()
	for k := range query {
		// setting $arg_{query_name}
		m[vArg+"_"+k] = query.Get(k)
	}
	m[vArgs] = r.URL.RawQuery
	m[vContentLength] = r.Header.Get("Content-Length")
	m[vContentType] = r.Header.Get("Content-Type")
	for _, cookie := range r.Cookies() {
		m[vCookie+"_"+cookie.Name] = cookie.Value
	}
	ctx := r.Context()
	root := ""
	if v := ctx.Value(rootKey{}); v != nil {
		root = v.(string)
	} else if v := ctx.Value(aliasKey{}); v != nil {
		root = v.(string)
	}
	m[vDocumentRoot] = root
	m[vDocumentURI] = r.URL
	m[vURI] = r.URL
	host := r.Host
	if host == "" {
		host = r.Header.Get("Host")
	}
	if host == "" {
		if v := ctx.Value(serverNameKey{}); v != nil {
			host = v.(string)
		}
	}
	m[vHost] = host
	n, _ := os.Hostname() //TODO: handle errors
	m[vHostname] = n
	for h := range r.Header {
		m[vHTTP+"_"+h] = r.Header.Get(h)
	}
	a := ""
	if len(query) > 0 {
		a = "?"
	}
	m[vIsArgs] = a
}
