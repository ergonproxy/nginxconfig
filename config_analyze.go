package main

import (
	"fmt"
	"strings"
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
	NGXDirectConf     = 0x00010000  // main file (not used)
	NGXMainConf       = 0x00040000  // main context
	NGXEventConf      = 0x00080000  // events
	NGXMailMainConf   = 0x00100000  // mail
	NGXMailSrvConf    = 0x00200000  // mail > server
	NGXStreamMainConf = 0x00400000  // stream
	NGXStreamSrvConf  = 0x00800000  // stream > server
	NGXStreamUpsConf  = 0x01000000  // stream > upstream
	NGXHttpMainConf   = 0x02000000  // http
	NGXHttpSrvConf    = 0x04000000  // http > server
	NGXHttpLocConf    = 0x08000000  // http > location
	NGXHttpUpsConf    = 0x10000000  // http > upstream
	NGXHttpSifConf    = 0x20000000  // http > server > if
	NGXHttpLifConf    = 0x40000000  // http > location > if
	NGXHttpLmtConf    = 0x80000000  // http > location > limit_except
	NGXHttpOauth2Conf = 0x100000000 // http > oauth2

	NGXAnyConf = (NGXMainConf | NGXEventConf | NGXMailMainConf | NGXMailSrvConf |
		NGXStreamMainConf | NGXStreamSrvConf | NGXStreamUpsConf |
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpUpsConf | NGXHttpOauth2Conf)
)

var directives = map[string][]int{
	"absolute_redirect": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"accept_mutex": []int{
		NGXEventConf | NGXConfFlag},
	"accept_mutex_delay": []int{
		NGXEventConf | NGXConfTake1},
	"access_log": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXHttpLmtConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"add_after_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"add_before_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"add_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake23},
	"add_trailer": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake23},
	"addition_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"aio": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"aio_write": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"alias": []int{
		NGXHttpLocConf | NGXConfTake1},
	"allow": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ancient_browser": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"ancient_browser_value": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"auth_basic": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1},
	"auth_basic_user_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1},
	"auth_http": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"auth_http_header": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake2},
	"auth_http_pass_client_cert": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag},
	"auth_http_timeout": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"auth_request": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"auth_request_set": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"autoindex": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"autoindex_exact_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"autoindex_format": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"autoindex_localtime": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"break": []int{
		NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfNoArgs},
	"charset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"charset_map": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake2},
	"charset_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"chunked_transfer_encoding": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"client_body_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"client_body_in_file_only": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"client_body_in_single_buffer": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"client_body_temp_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234},
	"client_body_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"client_header_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"client_header_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"client_max_body_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"connection_pool_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"create_full_put_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"daemon": []int{
		NGXMainConf | NGXDirectConf | NGXConfFlag},
	"dav_access": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"dav_methods": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"debug_connection": []int{
		NGXEventConf | NGXConfTake1},
	"debug_points": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"default_type": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"deny": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"directio": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"directio_alignment": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"disable_symlinks": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"empty_gif": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"env": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"error_log": []int{
		NGXMainConf | NGXConf1More,
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More,
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"error_page": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConf2More},
	"etag": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"events": []int{
		NGXMainConf | NGXConfBlock | NGXConfNoArgs},
	"expires": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake12},
	"fastcgi_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"fastcgi_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"fastcgi_busy_buffers_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_background_update": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_cache_bypass": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_cache_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_lock": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_cache_lock_age": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_lock_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_max_range_offset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_methods": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_cache_min_uses": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_path": []int{
		NGXHttpMainConf | NGXConf2More},
	"fastcgi_cache_revalidate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_cache_use_stale": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_cache_valid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_catch_stderr": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_force_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_hide_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_ignore_client_abort": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_ignore_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_index": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_intercept_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_keep_conn": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_max_temp_file_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_no_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"fastcgi_param": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake23},
	"fastcgi_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"fastcgi_pass_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_pass_request_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_pass_request_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_request_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_send_lowat": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"fastcgi_split_path_info": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_store": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_store_access": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"fastcgi_temp_file_write_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_temp_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234},
	"flv": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"geo": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake12,
		NGXStreamMainConf | NGXConfBlock | NGXConfTake12},
	"geoip_city": []int{
		NGXHttpMainConf | NGXConfTake12,
		NGXStreamMainConf | NGXConfTake12},
	"geoip_country": []int{
		NGXHttpMainConf | NGXConfTake12,
		NGXStreamMainConf | NGXConfTake12},
	"geoip_org": []int{
		NGXHttpMainConf | NGXConfTake12,
		NGXStreamMainConf | NGXConfTake12},
	"geoip_proxy": []int{
		NGXHttpMainConf | NGXConfTake1},
	"geoip_proxy_recursive": []int{
		NGXHttpMainConf | NGXConfFlag},
	"google_perftools_profiles": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"grpc_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"grpc_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_hide_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ignore_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"grpc_intercept_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"grpc_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"grpc_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"grpc_pass_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_set_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"grpc_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"grpc_ssl_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_certificate_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_ciphers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_crl": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_password_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_protocols": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"grpc_ssl_server_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"grpc_ssl_session_reuse": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"grpc_ssl_trusted_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"grpc_ssl_verify": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"grpc_ssl_verify_depth": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"gunzip": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"gunzip_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"gzip": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"gzip_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"gzip_comp_level": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"gzip_disable": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"gzip_http_version": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"gzip_min_length": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"gzip_proxied": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"gzip_static": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"gzip_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"gzip_vary": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"hash": []int{
		NGXHttpUpsConf | NGXConfTake12,
		NGXStreamUpsConf | NGXConfTake12},
	"http": []int{
		NGXMainConf | NGXConfBlock | NGXConfNoArgs},
	"http2_body_preread_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_chunk_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"http2_idle_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_max_concurrent_pushes": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_max_concurrent_streams": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_max_field_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_max_header_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_max_requests": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"http2_push": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"http2_push_preload": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"http2_recv_buffer_size": []int{
		NGXHttpMainConf | NGXConfTake1},
	"http2_recv_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"if": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConfBlock | NGXConf1More},
	"if_modified_since": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"ignore_invalid_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"image_filter": []int{
		NGXHttpLocConf | NGXConfTake123},
	"image_filter_buffer": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"image_filter_interlace": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"image_filter_jpeg_quality": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"image_filter_sharpen": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"image_filter_transparency": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"image_filter_webp_quality": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"imap_auth": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"imap_capabilities": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"imap_client_buffer": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"include": []int{
		NGXAnyConf | NGXConfTake1},
	"index": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"internal": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"ip_hash": []int{
		NGXHttpUpsConf | NGXConfNoArgs},
	"keepalive": []int{
		NGXHttpUpsConf | NGXConfTake1},
	"keepalive_disable": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"keepalive_requests": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXHttpUpsConf | NGXConfTake1},
	"keepalive_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12,
		NGXHttpUpsConf | NGXConfTake1},
	"large_client_header_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake2},
	"least_conn": []int{
		NGXHttpUpsConf | NGXConfNoArgs,
		NGXStreamUpsConf | NGXConfNoArgs},
	"limit_conn": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake2},
	"limit_conn_log_level": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"limit_conn_status": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"limit_conn_zone": []int{
		NGXHttpMainConf | NGXConfTake2,
		NGXStreamMainConf | NGXConfTake2},
	"limit_except": []int{
		NGXHttpLocConf | NGXConfBlock | NGXConf1More},
	"limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"limit_rate_after": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"limit_req": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"limit_req_log_level": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"limit_req_status": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"limit_req_zone": []int{
		NGXHttpMainConf | NGXConfTake3},
	"lingering_close": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"lingering_time": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"lingering_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"listen": []int{
		NGXHttpSrvConf | NGXConf1More,
		NGXMailSrvConf | NGXConf1More,
		NGXStreamSrvConf | NGXConf1More},
	"load_module": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"location": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConfBlock | NGXConfTake12},
	"lock_file": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"log_format": []int{
		NGXHttpMainConf | NGXConf2More,
		NGXStreamMainConf | NGXConf2More},
	"log_not_found": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"log_subrequest": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"mail": []int{
		NGXMainConf | NGXConfBlock | NGXConfNoArgs},
	"map": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake2,
		NGXStreamMainConf | NGXConfBlock | NGXConfTake2},
	"map_hash_bucket_size": []int{
		NGXHttpMainConf | NGXConfTake1,
		NGXStreamMainConf | NGXConfTake1},
	"map_hash_max_size": []int{
		NGXHttpMainConf | NGXConfTake1,
		NGXStreamMainConf | NGXConfTake1},
	"master_process": []int{
		NGXMainConf | NGXDirectConf | NGXConfFlag},
	"max_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"memcached_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_gzip_flag": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"memcached_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"memcached_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"memcached_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"merge_slashes": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"min_delete_depth": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"mirror": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"mirror_request_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"modern_browser": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"modern_browser_value": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"mp4": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"mp4_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"mp4_max_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"msie_padding": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"msie_refresh": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"multi_accept": []int{
		NGXEventConf | NGXConfFlag},
	"open_file_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"open_file_cache_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"open_file_cache_min_uses": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"open_file_cache_valid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"open_log_file_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1234},
	"oauth2_redirect_separator":   []int{},
	"oauth2_authorization_expire": []int{},
	"oauth2_access_expire":        []int{},
	"oauth2_allow_get_access":     []int{},
	"oauth2_allowed_access_type":  []int{},
	"oauth2_token_type":           []int{},
	"oauth2_provider_name":        []int{},
	"oauth2_auth_endpoint":        []int{},
	"oauth2_token_endpoint":       []int{},
	"oauth2_info_endpoint":        []int{},
	"oauth2_session_path":         []int{},
	"oauth2_session_max_age":      []int{},
	"oauth2_session_domain":       []int{},
	"oauth2_session_secure":       []int{},
	"oauth2_session_hhhponly":     []int{},
	"oauth2_session_name":         []int{},
	"oauth2_csrf_secret":          []int{},
	"output_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"override_charset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"pcre_jit": []int{
		NGXMainConf | NGXDirectConf | NGXConfFlag},
	"perl": []int{
		NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1},
	"perl_modules": []int{
		NGXHttpMainConf | NGXConfTake1},
	"perl_require": []int{
		NGXHttpMainConf | NGXConfTake1},
	"perl_set": []int{
		NGXHttpMainConf | NGXConfTake2},
	"pid": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"pop3_auth": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"pop3_capabilities": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"port_in_redirect": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"postpone_output": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"preread_buffer_size": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"preread_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"protocol": []int{
		NGXMailSrvConf | NGXConfTake1},
	"proxy_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake12},
	"proxy_buffer": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"proxy_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"proxy_busy_buffers_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_background_update": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_cache_bypass": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_cache_convert_head": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_cache_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_lock": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_cache_lock_age": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_lock_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_max_range_offset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_methods": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_cache_min_uses": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_cache_path": []int{
		NGXHttpMainConf | NGXConf2More},
	"proxy_cache_revalidate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_cache_use_stale": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_cache_valid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_cookie_domain": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"proxy_cookie_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"proxy_download_rate": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_force_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_headers_hash_bucket_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_headers_hash_max_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_hide_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_http_version": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_ignore_client_abort": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_ignore_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_intercept_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_max_temp_file_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_method": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_no_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"proxy_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXHttpLmtConf | NGXConfTake1,
		NGXStreamSrvConf | NGXConfTake1},
	"proxy_pass_error_message": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag},
	"proxy_pass_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_pass_request_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_pass_request_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_protocol": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_protocol_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_redirect": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"proxy_request_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"proxy_requests": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_responses": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_send_lowat": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_set_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_set_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"proxy_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_ssl": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_ssl_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_certificate_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_ciphers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_crl": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_password_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_protocols": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"proxy_ssl_server_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_ssl_session_reuse": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_ssl_trusted_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_ssl_verify": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"proxy_ssl_verify_depth": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_store": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_store_access": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"proxy_temp_file_write_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"proxy_temp_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234},
	"proxy_timeout": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"proxy_upload_rate": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"random": []int{
		NGXHttpUpsConf | NGXConfNoArgs | NGXConfTake12,
		NGXStreamUpsConf | NGXConfNoArgs | NGXConfTake12},
	"random_index": []int{
		NGXHttpLocConf | NGXConfFlag},
	"read_ahead": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"real_ip_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"real_ip_recursive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"recursive_error_pages": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"referer_hash_bucket_size": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"referer_hash_max_size": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"request_pool_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"reset_timedout_connection": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"resolver": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More,
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"resolver_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"return": []int{
		NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake12,
		NGXStreamSrvConf | NGXConfTake1},
	"rewrite": []int{
		NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake23},
	"rewrite_log": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"root": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"satisfy": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"scgi_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"scgi_busy_buffers_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_background_update": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_cache_bypass": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_cache_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_lock": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_cache_lock_age": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_lock_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_max_range_offset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_methods": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_cache_min_uses": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_cache_path": []int{
		NGXHttpMainConf | NGXConf2More},
	"scgi_cache_revalidate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_cache_use_stale": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_cache_valid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_force_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_hide_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_ignore_client_abort": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_ignore_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_intercept_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_max_temp_file_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_no_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"scgi_param": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake23},
	"scgi_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"scgi_pass_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_pass_request_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_pass_request_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_request_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"scgi_store": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_store_access": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"scgi_temp_file_write_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"scgi_temp_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234},
	"secure_link": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"secure_link_md5": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"secure_link_secret": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"send_lowat": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"sendfile": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"sendfile_max_chunk": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"server": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfNoArgs,
		NGXHttpUpsConf | NGXConf1More,
		NGXMailMainConf | NGXConfBlock | NGXConfNoArgs,
		NGXStreamMainConf | NGXConfBlock | NGXConfNoArgs,
		NGXStreamUpsConf | NGXConf1More},
	"server_name": []int{
		NGXHttpSrvConf | NGXConf1More,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"server_name_in_redirect": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"server_names_hash_bucket_size": []int{
		NGXHttpMainConf | NGXConfTake1},
	"server_names_hash_max_size": []int{
		NGXHttpMainConf | NGXConfTake1},
	"server_tokens": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"set": []int{
		NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake2},
	"set_real_ip_from": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"slice": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"smtp_auth": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"smtp_capabilities": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More},
	"smtp_client_buffer": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"smtp_greeting_delay": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"source_charset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"spdy_chunk_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"spdy_headers_comp": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"split_clients": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake2,
		NGXStreamMainConf | NGXConfBlock | NGXConfTake2},
	"ssi": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"ssi_last_modified": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"ssi_min_file_chunk": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"ssi_silent_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"ssi_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"ssi_value_length": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"ssl": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag,
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag},
	"ssl_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"ssl_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_certificate_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_ciphers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_client_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_crl": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_dhparam": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_early_data": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"ssl_ecdh_curve": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_engine": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"ssl_handshake_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_password_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_prefer_server_ciphers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag,
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"ssl_preread": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"ssl_protocols": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConf1More,
		NGXMailMainConf | NGXMailSrvConf | NGXConf1More,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"ssl_session_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake12,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake12,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake12},
	"ssl_session_ticket_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_session_tickets": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag,
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"ssl_session_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_stapling": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"ssl_stapling_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"ssl_stapling_responder": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1},
	"ssl_stapling_verify": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"ssl_trusted_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_verify_client": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"ssl_verify_depth": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfTake1,
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"starttls": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"stream": []int{
		NGXMainConf | NGXConfBlock | NGXConfNoArgs},
	"stub_status": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConfNoArgs | NGXConfTake1},
	"sub_filter": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"sub_filter_last_modified": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"sub_filter_once": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"sub_filter_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"subrequest_output_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"tcp_nodelay": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag,
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"tcp_nopush": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"thread_pool": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake23},
	"timeout": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfTake1},
	"timer_resolution": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"try_files": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConf2More},
	"types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfBlock | NGXConfNoArgs},
	"types_hash_bucket_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"types_hash_max_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"underscores_in_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXConfFlag},
	"uninitialized_variable_warn": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpSifConf | NGXHttpLocConf | NGXHttpLifConf | NGXConfFlag},
	"upstream": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake1,
		NGXStreamMainConf | NGXConfBlock | NGXConfTake1},
	"use": []int{
		NGXEventConf | NGXConfTake1},
	"user": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake12},
	"userid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_domain": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_expires": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_mark": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_p3p": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"userid_service": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_bind": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"uwsgi_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"uwsgi_busy_buffers_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_background_update": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_bypass": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_cache_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_lock": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_cache_lock_age": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_lock_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_max_range_offset": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_methods": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_cache_min_uses": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_cache_path": []int{
		NGXHttpMainConf | NGXConf2More},
	"uwsgi_cache_revalidate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_cache_use_stale": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_cache_valid": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_connect_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_force_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_hide_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ignore_client_abort": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_ignore_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_intercept_errors": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_max_temp_file_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_modifier1": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_modifier2": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_next_upstream": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_next_upstream_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_next_upstream_tries": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_no_cache": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_param": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake23},
	"uwsgi_pass": []int{
		NGXHttpLocConf | NGXHttpLifConf | NGXConfTake1},
	"uwsgi_pass_header": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_pass_request_body": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_pass_request_headers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_read_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_request_buffering": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_send_timeout": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_socket_keepalive": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_ssl_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_certificate_key": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_ciphers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_crl": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_password_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_protocols": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"uwsgi_ssl_server_name": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_ssl_session_reuse": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_ssl_trusted_certificate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_ssl_verify": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"uwsgi_ssl_verify_depth": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_store": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_store_access": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake123},
	"uwsgi_temp_file_write_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"uwsgi_temp_path": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1234},
	"valid_referers": []int{
		NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"variables_hash_bucket_size": []int{
		NGXHttpMainConf | NGXConfTake1,
		NGXStreamMainConf | NGXConfTake1},
	"variables_hash_max_size": []int{
		NGXHttpMainConf | NGXConfTake1,
		NGXStreamMainConf | NGXConfTake1},
	"worker_aio_requests": []int{
		NGXEventConf | NGXConfTake1},
	"worker_connections": []int{
		NGXEventConf | NGXConfTake1},
	"worker_cpu_affinity": []int{
		NGXMainConf | NGXDirectConf | NGXConf1More},
	"worker_priority": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"worker_processes": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"worker_rlimit_core": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"worker_rlimit_nofile": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"worker_shutdown_timeout": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"working_directory": []int{
		NGXMainConf | NGXDirectConf | NGXConfTake1},
	"xclient": []int{
		NGXMailMainConf | NGXMailSrvConf | NGXConfFlag},
	"xml_entities": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"xslt_last_modified": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"xslt_param": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"xslt_string_param": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"xslt_stylesheet": []int{
		NGXHttpLocConf | NGXConf1More},
	"xslt_types": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"zone": []int{
		NGXHttpUpsConf | NGXConfTake12,
		NGXStreamUpsConf | NGXConfTake12},

	// nginx+ directives [definitions inferred from docs]
	"api": []int{
		NGXHttpLocConf | NGXConfNoArgs | NGXConfTake1},
	"auth_jwt": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"auth_jwt_claim_set": []int{
		NGXHttpMainConf | NGXConf2More},
	"auth_jwt_header_set": []int{
		NGXHttpMainConf | NGXConf2More},
	"auth_jwt_key_file": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"auth_jwt_key_request": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"auth_jwt_leeway": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"f4f": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"f4f_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"fastcgi_cache_purge": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"health_check": []int{
		NGXHttpLocConf | NGXConfAny,
		NGXStreamSrvConf | NGXConfAny},
	"health_check_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"hls": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"hls_buffers": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake2},
	"hls_forward_args": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"hls_fragment": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"hls_mp4_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"hls_mp4_max_buffer_size": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"js_access": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"js_content": []int{
		NGXHttpLocConf | NGXHttpLmtConf | NGXConfTake1},
	"js_filter": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"js_include": []int{
		NGXHttpMainConf | NGXConfTake1,
		NGXStreamMainConf | NGXConfTake1},
	"js_path": []int{
		NGXHttpMainConf | NGXConfTake1},
	"js_preread": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"js_set": []int{
		NGXHttpMainConf | NGXConfTake2,
		NGXStreamMainConf | NGXConfTake2},
	"keyval": []int{
		NGXHttpMainConf | NGXConfTake3,
		NGXStreamMainConf | NGXConfTake3},
	"keyval_zone": []int{
		NGXHttpMainConf | NGXConf1More,
		NGXStreamMainConf | NGXConf1More},
	"least_time": []int{
		NGXHttpUpsConf | NGXConfTake12,
		NGXStreamUpsConf | NGXConfTake12},
	"limit_zone": []int{
		NGXHttpMainConf | NGXConfTake3},
	"match": []int{
		NGXHttpMainConf | NGXConfBlock | NGXConfTake1,
		NGXStreamMainConf | NGXConfBlock | NGXConfTake1},
	"memcached_force_ranges": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfFlag},
	"mp4_limit_rate": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"mp4_limit_rate_after": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"ntlm": []int{
		NGXHttpUpsConf | NGXConfNoArgs},
	"proxy_cache_purge": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"queue": []int{
		NGXHttpUpsConf | NGXConfTake12},
	"scgi_cache_purge": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"session_log": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake1},
	"session_log_format": []int{
		NGXHttpMainConf | NGXConf2More},
	"session_log_zone": []int{
		NGXHttpMainConf | NGXConfTake23 | NGXConfTake4 | NGXConfTake5 | NGXConfTake6},
	"state": []int{
		NGXHttpUpsConf | NGXConfTake1,
		NGXStreamUpsConf | NGXConfTake1},
	"status": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"status_format": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConfTake12},
	"status_zone": []int{
		NGXHttpSrvConf | NGXConfTake1,
		NGXStreamSrvConf | NGXConfTake1,
		NGXHttpLocConf | NGXConfTake1,
		NGXHttpLifConf | NGXConfTake1},
	"sticky": []int{
		NGXHttpUpsConf | NGXConf1More},
	"sticky_cookie_insert": []int{
		NGXHttpUpsConf | NGXConfTake1234},
	"upstream_conf": []int{
		NGXHttpLocConf | NGXConfNoArgs},
	"uwsgi_cache_purge": []int{
		NGXHttpMainConf | NGXHttpSrvConf | NGXHttpLocConf | NGXConf1More},
	"zone_sync": []int{
		NGXStreamSrvConf | NGXConfNoArgs},
	"zone_sync_buffers": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake2},
	"zone_sync_connect_retry_interval": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_connect_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_interval": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_recv_buffer_size": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_server": []int{
		NGXStreamSrvConf | NGXConfTake12},
	"zone_sync_ssl": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"zone_sync_ssl_certificate": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_certificate_key": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_ciphers": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_crl": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_name": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_password_file": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_protocols": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConf1More},
	"zone_sync_ssl_server_name": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"zone_sync_ssl_trusted_certificate": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_ssl_verify": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfFlag},
	"zone_sync_ssl_verify_depth": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
	"zone_sync_timeout": []int{
		NGXStreamMainConf | NGXStreamSrvConf | NGXConfTake1},
}

var contexts = map[string]int{
	toCtx():                                   NGXMainConf,
	toCtx("events"):                           NGXEventConf,
	toCtx("mail"):                             NGXMailMainConf,
	toCtx("mail", "server"):                   NGXMailSrvConf,
	toCtx("stream"):                           NGXStreamMainConf,
	toCtx("stream", "server"):                 NGXStreamSrvConf,
	toCtx("stream", "upstream"):               NGXStreamUpsConf,
	toCtx("http"):                             NGXHttpMainConf,
	toCtx("http", "server"):                   NGXHttpSrvConf,
	toCtx("http", "location"):                 NGXHttpLocConf,
	toCtx("http", "upstream"):                 NGXHttpUpsConf,
	toCtx("http", "server", "if"):             NGXHttpSifConf,
	toCtx("http", "location", "if"):           NGXHttpLifConf,
	toCtx("http", "location", "limit_except"): NGXHttpLmtConf,
	toCtx("http", "oauth2"):                   NGXHttpOauth2Conf,
}

func toCtx(s ...string) string {
	if len(s) > 0 {
		return strings.Join(s, ",")
	}
	return ""
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
