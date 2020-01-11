package main

import (
	"fmt"
	"strings"

	"github.com/ergongate/vince/config/nginx"
)

// bit masks for different directive argument styles
const (
	NGXConfNoArgs = 0x00000001 // 0 args
	NGXConfTake1  = 0x00000002 // 1 args
	NGXConfTake2  = 0x00000004 // 2 args
	NGXConfTake3  = 0x00000008 // 3 args
	NGXConfTake4  = 0x00000010 // 4 args
	NGXConfTake5  = 0x00000020 // 5 args
	NGXConfTake6  = 0x00000040 // 6 args
	NGXConfTake7  = 0x00000080 // 7 args
	NGXConfBlock  = 0x00000100 // followed by block
	NGXConfFlag   = 0x00000200 // 'on' or 'off'
	NGXConfAny    = 0x00000400 // >=0 args
	NGXConf1More  = 0x00000800 // >=1 args
	NGXConf2More  = 0x00001000 // >=2 args

	// some helpful argument style aliases
	NGXConfTake12   = (NGXConfTake1 | NGXConfTake2)
	NGXConfTake13   = (NGXConfTake1 | NGXConfTake3)
	NGXConfTake23   = (NGXConfTake2 | NGXConfTake3)
	NGXConfTake123  = (NGXConfTake12 | NGXConfTake3)
	NGXConfTake1234 = (NGXConfTake123 | NGXConfTake4)

	// bit masks for different directive locations
	NGX_DIRECT_CONF      = 0x00010000 // main file (not used)
	NGX_MAIN_CONF        = 0x00040000 // main context
	NGX_EVENT_CONF       = 0x00080000 // events
	NGX_MAIL_MAIN_CONF   = 0x00100000 // mail
	NGX_MAIL_SRV_CONF    = 0x00200000 // mail > server
	NGX_STREAM_MAIN_CONF = 0x00400000 // stream
	NGX_STREAM_SRV_CONF  = 0x00800000 // stream > server
	NGX_STREAM_UPS_CONF  = 0x01000000 // stream > upstream
	NGX_HTTP_MAIN_CONF   = 0x02000000 // http
	NGX_HTTP_SRV_CONF    = 0x04000000 // http > server
	NGX_HTTP_LOC_CONF    = 0x08000000 // http > location
	NGX_HTTP_UPS_CONF    = 0x10000000 // http > upstream
	NGX_HTTP_SIF_CONF    = 0x20000000 // http > server > if
	NGX_HTTP_LIF_CONF    = 0x40000000 // http > location > if
	NGX_HTTP_LMT_CONF    = 0x80000000 // http > location > limit_except

	NGX_ANY_CONF = (NGX_MAIN_CONF | NGX_EVENT_CONF | NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF |
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGX_STREAM_UPS_CONF |
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_UPS_CONF)
)

var directives = map[string][]int{
	"absolute_redirect": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"accept_mutex": []int{
		NGX_EVENT_CONF | NGXConfFlag},
	"accept_mutex_delay": []int{
		NGX_EVENT_CONF | NGXConfTake1},
	"access_log": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGX_HTTP_LMT_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"add_after_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"add_before_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"add_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake23},
	"add_trailer": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake23},
	"addition_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"aio": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"aio_write": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"alias": []int{
		NGX_HTTP_LOC_CONF | NGXConfTake1},
	"allow": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ancient_browser": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"ancient_browser_value": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"auth_basic": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1},
	"auth_basic_user_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1},
	"auth_http": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"auth_http_header": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake2},
	"auth_http_pass_client_cert": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag},
	"auth_http_timeout": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"auth_request": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"auth_request_set": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"autoindex": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"autoindex_exact_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"autoindex_format": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"autoindex_localtime": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"break": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfNoArgs},
	"charset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"charset_map": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake2},
	"charset_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"chunked_transfer_encoding": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"client_body_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"client_body_in_file_only": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"client_body_in_single_buffer": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"client_body_temp_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234},
	"client_body_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"client_header_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"client_header_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"client_max_body_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"connection_pool_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"create_full_put_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"daemon": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfFlag},
	"dav_access": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"dav_methods": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"debug_connection": []int{
		NGX_EVENT_CONF | NGXConfTake1},
	"debug_points": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"default_type": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"deny": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"directio": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"directio_alignment": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"disable_symlinks": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"empty_gif": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"env": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"error_log": []int{
		NGX_MAIN_CONF | NGXConf1More,
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"error_page": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConf2More},
	"etag": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"events": []int{
		NGX_MAIN_CONF | NGXConfBlock | NGXConfNoArgs},
	"expires": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake12},
	"fastcgi_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"fastcgi_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"fastcgi_busy_buffers_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_background_update": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_cache_bypass": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_cache_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_lock": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_cache_lock_age": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_lock_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_max_range_offset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_methods": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_cache_min_uses": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_path": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"fastcgi_cache_revalidate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_cache_use_stale": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_cache_valid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_catch_stderr": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_force_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_hide_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_ignore_client_abort": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_ignore_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_index": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_intercept_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_keep_conn": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_max_temp_file_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_no_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"fastcgi_param": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake23},
	"fastcgi_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"fastcgi_pass_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_pass_request_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_pass_request_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_request_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_send_lowat": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"fastcgi_split_path_info": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_store": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_store_access": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"fastcgi_temp_file_write_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_temp_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234},
	"flv": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"geo": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfTake12},
	"geoip_city": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGXConfTake12},
	"geoip_country": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGXConfTake12},
	"geoip_org": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGXConfTake12},
	"geoip_proxy": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"geoip_proxy_recursive": []int{
		NGX_HTTP_MAIN_CONF | NGXConfFlag},
	"google_perftools_profiles": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"grpc_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"grpc_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_hide_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ignore_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"grpc_intercept_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"grpc_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"grpc_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"grpc_pass_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_set_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"grpc_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"grpc_ssl_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_certificate_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_ciphers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_crl": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_password_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_protocols": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"grpc_ssl_server_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"grpc_ssl_session_reuse": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"grpc_ssl_trusted_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"grpc_ssl_verify": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"grpc_ssl_verify_depth": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"gunzip": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"gunzip_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"gzip": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"gzip_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"gzip_comp_level": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"gzip_disable": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"gzip_http_version": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"gzip_min_length": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"gzip_proxied": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"gzip_static": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"gzip_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"gzip_vary": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"hash": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake12,
		NGX_STREAM_UPS_CONF | NGXConfTake12},
	"http": []int{
		NGX_MAIN_CONF | NGXConfBlock | NGXConfNoArgs},
	"http2_body_preread_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_chunk_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"http2_idle_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_max_concurrent_pushes": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_max_concurrent_streams": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_max_field_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_max_header_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_max_requests": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"http2_push": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"http2_push_preload": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"http2_recv_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"http2_recv_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"if": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfBlock | NGXConf1More},
	"if_modified_since": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"ignore_invalid_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"image_filter": []int{
		NGX_HTTP_LOC_CONF | NGXConfTake123},
	"image_filter_buffer": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"image_filter_interlace": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"image_filter_jpeg_quality": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"image_filter_sharpen": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"image_filter_transparency": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"image_filter_webp_quality": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"imap_auth": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"imap_capabilities": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"imap_client_buffer": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"include": []int{
		NGX_ANY_CONF | NGXConfTake1},
	"index": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"internal": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"ip_hash": []int{
		NGX_HTTP_UPS_CONF | NGXConfNoArgs},
	"keepalive": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake1},
	"keepalive_disable": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"keepalive_requests": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_HTTP_UPS_CONF | NGXConfTake1},
	"keepalive_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12,
		NGX_HTTP_UPS_CONF | NGXConfTake1},
	"large_client_header_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake2},
	"least_conn": []int{
		NGX_HTTP_UPS_CONF | NGXConfNoArgs,
		NGX_STREAM_UPS_CONF | NGXConfNoArgs},
	"limit_conn": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake2},
	"limit_conn_log_level": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"limit_conn_status": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"limit_conn_zone": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake2,
		NGX_STREAM_MAIN_CONF | NGXConfTake2},
	"limit_except": []int{
		NGX_HTTP_LOC_CONF | NGXConfBlock | NGXConf1More},
	"limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"limit_rate_after": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"limit_req": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"limit_req_log_level": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"limit_req_status": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"limit_req_zone": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake3},
	"lingering_close": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"lingering_time": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"lingering_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"listen": []int{
		NGX_HTTP_SRV_CONF | NGXConf1More,
		NGX_MAIL_SRV_CONF | NGXConf1More,
		NGX_STREAM_SRV_CONF | NGXConf1More},
	"load_module": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"location": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfBlock | NGXConfTake12},
	"lock_file": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"log_format": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More,
		NGX_STREAM_MAIN_CONF | NGXConf2More},
	"log_not_found": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"log_subrequest": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"mail": []int{
		NGX_MAIN_CONF | NGXConfBlock | NGXConfNoArgs},
	"map": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake2,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfTake2},
	"map_hash_bucket_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfTake1},
	"map_hash_max_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfTake1},
	"master_process": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfFlag},
	"max_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"memcached_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_gzip_flag": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"memcached_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"memcached_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"memcached_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"merge_slashes": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"min_delete_depth": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"mirror": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"mirror_request_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"modern_browser": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"modern_browser_value": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"mp4": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"mp4_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"mp4_max_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"msie_padding": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"msie_refresh": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"multi_accept": []int{
		NGX_EVENT_CONF | NGXConfFlag},
	"open_file_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"open_file_cache_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"open_file_cache_min_uses": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"open_file_cache_valid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"open_log_file_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1234},
	"output_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"override_charset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"pcre_jit": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfFlag},
	"perl": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1},
	"perl_modules": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"perl_require": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"perl_set": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake2},
	"pid": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"pop3_auth": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"pop3_capabilities": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"port_in_redirect": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"postpone_output": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"preread_buffer_size": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"preread_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"protocol": []int{
		NGX_MAIL_SRV_CONF | NGXConfTake1},
	"proxy_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake12},
	"proxy_buffer": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"proxy_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"proxy_busy_buffers_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_background_update": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_cache_bypass": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_cache_convert_head": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_cache_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_lock": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_cache_lock_age": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_lock_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_max_range_offset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_methods": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_cache_min_uses": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_cache_path": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"proxy_cache_revalidate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_cache_use_stale": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_cache_valid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_cookie_domain": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"proxy_cookie_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"proxy_download_rate": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_force_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_headers_hash_bucket_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_headers_hash_max_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_hide_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_http_version": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_ignore_client_abort": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_ignore_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_intercept_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_max_temp_file_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_method": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_no_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"proxy_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1,
		NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_pass_error_message": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag},
	"proxy_pass_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_pass_request_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_pass_request_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_protocol": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_protocol_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_redirect": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"proxy_request_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"proxy_requests": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_responses": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_send_lowat": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_set_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_set_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"proxy_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_ssl": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_ssl_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_certificate_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_ciphers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_crl": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_password_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_protocols": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"proxy_ssl_server_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_ssl_session_reuse": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_ssl_trusted_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_ssl_verify": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"proxy_ssl_verify_depth": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_store": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_store_access": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"proxy_temp_file_write_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"proxy_temp_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234},
	"proxy_timeout": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"proxy_upload_rate": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"random": []int{
		NGX_HTTP_UPS_CONF | NGXConfNoArgs | NGXConfTake12,
		NGX_STREAM_UPS_CONF | NGXConfNoArgs | NGXConfTake12},
	"random_index": []int{
		NGX_HTTP_LOC_CONF | NGXConfFlag},
	"read_ahead": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"real_ip_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"real_ip_recursive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"recursive_error_pages": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"referer_hash_bucket_size": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"referer_hash_max_size": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"request_pool_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"reset_timedout_connection": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"resolver": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"resolver_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"return": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake12,
		NGX_STREAM_SRV_CONF | NGXConfTake1},
	"rewrite": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake23},
	"rewrite_log": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"root": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"satisfy": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"scgi_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"scgi_busy_buffers_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_background_update": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_cache_bypass": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_cache_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_lock": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_cache_lock_age": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_lock_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_max_range_offset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_methods": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_cache_min_uses": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_cache_path": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"scgi_cache_revalidate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_cache_use_stale": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_cache_valid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_force_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_hide_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_ignore_client_abort": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_ignore_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_intercept_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_max_temp_file_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_no_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"scgi_param": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake23},
	"scgi_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"scgi_pass_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_pass_request_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_pass_request_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_request_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"scgi_store": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_store_access": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"scgi_temp_file_write_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"scgi_temp_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234},
	"secure_link": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"secure_link_md5": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"secure_link_secret": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"send_lowat": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"sendfile": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"sendfile_max_chunk": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"server": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfNoArgs,
		NGX_HTTP_UPS_CONF | NGXConf1More,
		NGX_MAIL_MAIN_CONF | NGXConfBlock | NGXConfNoArgs,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfNoArgs,
		NGX_STREAM_UPS_CONF | NGXConf1More},
	"server_name": []int{
		NGX_HTTP_SRV_CONF | NGXConf1More,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"server_name_in_redirect": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"server_names_hash_bucket_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"server_names_hash_max_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"server_tokens": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"set": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake2},
	"set_real_ip_from": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"slice": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"smtp_auth": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"smtp_capabilities": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More},
	"smtp_client_buffer": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"smtp_greeting_delay": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"source_charset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"spdy_chunk_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"spdy_headers_comp": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"split_clients": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake2,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfTake2},
	"ssi": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"ssi_last_modified": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"ssi_min_file_chunk": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"ssi_silent_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"ssi_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"ssi_value_length": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"ssl": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag},
	"ssl_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"ssl_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_certificate_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_ciphers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_client_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_crl": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_dhparam": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_early_data": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"ssl_ecdh_curve": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_engine": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"ssl_handshake_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_password_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_prefer_server_ciphers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"ssl_preread": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"ssl_protocols": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConf1More,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"ssl_session_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake12,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake12,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake12},
	"ssl_session_ticket_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_session_tickets": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"ssl_session_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_stapling": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"ssl_stapling_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"ssl_stapling_responder": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1},
	"ssl_stapling_verify": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"ssl_trusted_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_verify_client": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"ssl_verify_depth": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"starttls": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"stream": []int{
		NGX_MAIN_CONF | NGXConfBlock | NGXConfNoArgs},
	"stub_status": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfNoArgs | NGXConfTake1},
	"sub_filter": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"sub_filter_last_modified": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"sub_filter_once": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"sub_filter_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"subrequest_output_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"tcp_nodelay": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag,
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"tcp_nopush": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"thread_pool": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake23},
	"timeout": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfTake1},
	"timer_resolution": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"try_files": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf2More},
	"types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfBlock | NGXConfNoArgs},
	"types_hash_bucket_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"types_hash_max_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"underscores_in_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGXConfFlag},
	"uninitialized_variable_warn": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_SIF_CONF | NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfFlag},
	"upstream": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfTake1},
	"use": []int{
		NGX_EVENT_CONF | NGXConfTake1},
	"user": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake12},
	"userid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_domain": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_expires": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_mark": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_p3p": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"userid_service": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_bind": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"uwsgi_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"uwsgi_busy_buffers_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_background_update": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_bypass": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_cache_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_lock": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_cache_lock_age": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_lock_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_max_range_offset": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_methods": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_cache_min_uses": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_cache_path": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"uwsgi_cache_revalidate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_cache_use_stale": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_cache_valid": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_connect_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_force_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_hide_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ignore_client_abort": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_ignore_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_intercept_errors": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_max_temp_file_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_modifier1": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_modifier2": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_next_upstream": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_next_upstream_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_next_upstream_tries": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_no_cache": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_param": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake23},
	"uwsgi_pass": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LIF_CONF | NGXConfTake1},
	"uwsgi_pass_header": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_pass_request_body": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_pass_request_headers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_read_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_request_buffering": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_send_timeout": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_socket_keepalive": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_ssl_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_certificate_key": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_ciphers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_crl": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_password_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_protocols": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"uwsgi_ssl_server_name": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_ssl_session_reuse": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_ssl_trusted_certificate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_ssl_verify": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"uwsgi_ssl_verify_depth": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_store": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_store_access": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake123},
	"uwsgi_temp_file_write_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"uwsgi_temp_path": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1234},
	"valid_referers": []int{
		NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"variables_hash_bucket_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfTake1},
	"variables_hash_max_size": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfTake1},
	"worker_aio_requests": []int{
		NGX_EVENT_CONF | NGXConfTake1},
	"worker_connections": []int{
		NGX_EVENT_CONF | NGXConfTake1},
	"worker_cpu_affinity": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConf1More},
	"worker_priority": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"worker_processes": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"worker_rlimit_core": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"worker_rlimit_nofile": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"worker_shutdown_timeout": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"working_directory": []int{
		NGX_MAIN_CONF | NGX_DIRECT_CONF | NGXConfTake1},
	"xclient": []int{
		NGX_MAIL_MAIN_CONF | NGX_MAIL_SRV_CONF | NGXConfFlag},
	"xml_entities": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"xslt_last_modified": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"xslt_param": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"xslt_string_param": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"xslt_stylesheet": []int{
		NGX_HTTP_LOC_CONF | NGXConf1More},
	"xslt_types": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"zone": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake12,
		NGX_STREAM_UPS_CONF | NGXConfTake12},

	// nginx+ directives [definitions inferred from docs]
	"api": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs | NGXConfTake1},
	"auth_jwt": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"auth_jwt_claim_set": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"auth_jwt_header_set": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"auth_jwt_key_file": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"auth_jwt_key_request": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"auth_jwt_leeway": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"f4f": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"f4f_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"fastcgi_cache_purge": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"health_check": []int{
		NGX_HTTP_LOC_CONF | NGXConfAny,
		NGX_STREAM_SRV_CONF | NGXConfAny},
	"health_check_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"hls": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"hls_buffers": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake2},
	"hls_forward_args": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"hls_fragment": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"hls_mp4_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"hls_mp4_max_buffer_size": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"js_access": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"js_content": []int{
		NGX_HTTP_LOC_CONF | NGX_HTTP_LMT_CONF | NGXConfTake1},
	"js_filter": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"js_include": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfTake1},
	"js_path": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake1},
	"js_preread": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"js_set": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake2,
		NGX_STREAM_MAIN_CONF | NGXConfTake2},
	"keyval": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake3,
		NGX_STREAM_MAIN_CONF | NGXConfTake3},
	"keyval_zone": []int{
		NGX_HTTP_MAIN_CONF | NGXConf1More,
		NGX_STREAM_MAIN_CONF | NGXConf1More},
	"least_time": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake12,
		NGX_STREAM_UPS_CONF | NGXConfTake12},
	"limit_zone": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake3},
	"match": []int{
		NGX_HTTP_MAIN_CONF | NGXConfBlock | NGXConfTake1,
		NGX_STREAM_MAIN_CONF | NGXConfBlock | NGXConfTake1},
	"memcached_force_ranges": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfFlag},
	"mp4_limit_rate": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"mp4_limit_rate_after": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"ntlm": []int{
		NGX_HTTP_UPS_CONF | NGXConfNoArgs},
	"proxy_cache_purge": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"queue": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake12},
	"scgi_cache_purge": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"session_log": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake1},
	"session_log_format": []int{
		NGX_HTTP_MAIN_CONF | NGXConf2More},
	"session_log_zone": []int{
		NGX_HTTP_MAIN_CONF | NGXConfTake23 | NGXConfTake4 | NGXConfTake5 | NGXConfTake6},
	"state": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake1,
		NGX_STREAM_UPS_CONF | NGXConfTake1},
	"status": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"status_format": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConfTake12},
	"status_zone": []int{
		NGX_HTTP_SRV_CONF | NGXConfTake1,
		NGX_STREAM_SRV_CONF | NGXConfTake1,
		NGX_HTTP_LOC_CONF | NGXConfTake1,
		NGX_HTTP_LIF_CONF | NGXConfTake1},
	"sticky": []int{
		NGX_HTTP_UPS_CONF | NGXConf1More},
	"sticky_cookie_insert": []int{
		NGX_HTTP_UPS_CONF | NGXConfTake1234},
	"upstream_conf": []int{
		NGX_HTTP_LOC_CONF | NGXConfNoArgs},
	"uwsgi_cache_purge": []int{
		NGX_HTTP_MAIN_CONF | NGX_HTTP_SRV_CONF | NGX_HTTP_LOC_CONF | NGXConf1More},
	"zone_sync": []int{
		NGX_STREAM_SRV_CONF | NGXConfNoArgs},
	"zone_sync_buffers": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake2},
	"zone_sync_connect_retry_interval": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_connect_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_interval": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_recv_buffer_size": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_server": []int{
		NGX_STREAM_SRV_CONF | NGXConfTake12},
	"zone_sync_ssl": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"zone_sync_ssl_certificate": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_certificate_key": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_ciphers": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_crl": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_name": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_password_file": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_protocols": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConf1More},
	"zone_sync_ssl_server_name": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"zone_sync_ssl_trusted_certificate": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_ssl_verify": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfFlag},
	"zone_sync_ssl_verify_depth": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
	"zone_sync_timeout": []int{
		NGX_STREAM_MAIN_CONF | NGX_STREAM_SRV_CONF | NGXConfTake1},
}

var contexts = map[string]int{
	toCtx():                                   NGX_MAIN_CONF,
	toCtx("events"):                           NGX_EVENT_CONF,
	toCtx("mail"):                             NGX_MAIL_MAIN_CONF,
	toCtx("mail", "server"):                   NGX_MAIL_SRV_CONF,
	toCtx("stream"):                           NGX_STREAM_MAIN_CONF,
	toCtx("stream", "server"):                 NGX_STREAM_SRV_CONF,
	toCtx("stream", "upstream"):               NGX_STREAM_UPS_CONF,
	toCtx("http"):                             NGX_HTTP_MAIN_CONF,
	toCtx("http", "server"):                   NGX_HTTP_SRV_CONF,
	toCtx("http", "location"):                 NGX_HTTP_LOC_CONF,
	toCtx("http", "upstream"):                 NGX_HTTP_UPS_CONF,
	toCtx("http", "server", "if"):             NGX_HTTP_SIF_CONF,
	toCtx("http", "location", "if"):           NGX_HTTP_LIF_CONF,
	toCtx("http", "location", "limit_except"): NGX_HTTP_LMT_CONF,
}

func toCtx(s ...string) string {
	if len(s) > 0 {
		return strings.Join(s, ",")
	}
	return ""
}

func enterBlockContext(stmt *nginx.Directive, ctx []string) []string {
	if len(ctx) > 0 && ctx[0] == "http" && stmt.Name == "location" {
		return []string{"http", "location"}
	}
	return append(ctx, stmt.Name)
}

func analyze(filename string, stmt *Stmt, term string, ctx []string, strict bool, checkCtx bool, checkArgs bool) error {
	masks, ok := directives[stmt.Directive]
	if strict && !ok {
		return &NgxParserDirectiveUnknownError{
			NgxError: newError(
				fmt.Sprintf("unknown directive %q", stmt.Directive),
				stmt.Line,
				filename,
			),
		}
	}
	ctxMask, ctxOk := contexts[toCtx(ctx...)]
	if !ctxOk || !ok {
		return nil
	}
	if checkCtx {
		pass := true
		for _, m := range masks {
			if m&ctxMask != 0 {
				pass = true
				break
			}
		}
		if !pass {
			return &NgxParserDirectiveUnknownError{
				NgxError: newError(
					fmt.Sprintf("%q directive is not allowed here", stmt.Directive),
					stmt.Line,
					filename,
				),
			}
		}
	}
	if !checkArgs {
		return nil
	}
	n := len(stmt.Args)
	var reason []string
	for _, mask := range reverse(masks) {
		if (mask&NGXConfBlock != 0) && term != "{" {
			reason = append(reason, "directive %q has no opening {")
			continue
		}
		if !(mask&NGXConfBlock != 0) && term != ";" {
			reason = append(reason, "directive %q is not terminated by ;")
			continue
		}
		if ((mask>>uint(n))&1 != 0 && n <= 7) ||
			(mask&NGXConfFlag != 0 && n == 1 && validFlag(stmt.Args[0])) ||
			(mask&NGXConfAny != 0 && n >= 0) ||
			(mask&NGXConf1More != 0 && n >= 1) ||
			(mask&NGXConf2More != 0 && n >= 2) {
			return nil
		} else if mask&NGXConfFlag != 0 && n == 1 && !validFlag(stmt.Args[0]) {
			reason = append(reason,
				"invalid value "+stmt.Args[0]+" in %s directive ,it must be on or off",
			)
		} else {
			reason = append(reason, "invalid number of arguments in %q directive")
		}
	}
	if len(reason) > 0 {
		for i := 0; i < len(reason); i++ {
			reason[i] = fmt.Sprintf(reason[0], stmt.Directive)
		}
		return &NgxParserDirectiveUnknownError{
			NgxError: newError(
				strings.Join(reason, ","),
				stmt.Line,
				filename,
			),
		}
	}
	return nil
}
func inDirective(name string) bool {
	_, ok := directives[name]
	return ok
}

func inContext(ctx []string) bool {
	_, ok := contexts[toCtx(ctx...)]
	return ok
}
func reverse(a []int) []int {
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}
	return a
}

func validFlag(s string) bool {
	switch strings.ToLower(s) {
	case "on", "off":
		return true
	default:
		return false
	}
}
