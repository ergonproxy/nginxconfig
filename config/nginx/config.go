package nginx

import (
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Events struct {
	AcceptMutex       bool          `nginx:"accept_mutex,off"`
	AcceptMutextDelay time.Duration `nginx:"accept_mutex_delay,500ms"`
	DebugConnection   []*Connection `nginx:"debug_connection"`
	MultiAccept       bool          `nginx:"master_process"`
	Use               string        `nginx:"use"`
	WorkerAIORequests int           `nginx:"worker_aio_requests"`
	WorkerConnections int           `nginx:"worker_connections"`
}

// Core configuration for core nginx functionality
type Core struct {
	Daemon                bool          `nginx:"daemon,off"`
	DebugPoints           string        `nginx:"debug_points"`
	Env                   []KV          `nginx:"env"`
	ErrorLog              []ErrorLog    `nginx:"error_log"`
	LockFile              string        `nginx:"lock_file"`
	MasterProcess         bool          `nginx:"master_process"`
	PCREJit               bool          `nginx:"pcre_jit"`
	PID                   string        `nginx:"pid"`
	SSLEngine             string        `nginx:"ssl_engine"`
	ThreadPool            *ThreadPool   `nginx:"thread_pool"`
	TimerResolution       string        `nginx:"timer_resolution"`
	User                  *User         `nginx:"user"`
	Events                *Events       `nginx:"events"`
	WorkerCPUAffinity     string        `nginx:"worker_cpu_affinity"`
	WorkerPriority        int           `nginx:"worker_priority"`
	WorkerProcess         int           `nginx:"worker_processes"`
	WorkerRLimitNoFile    int           `nginx:"worker_rlimit_nofile"`
	WorkerShutdownTimeout time.Duration `nginx:"worker_shutdown_timeout"`
	WorkingDirectory      string        `nginx:"working_directory"`
	LogSubrequests        bool          `nginx:"log_subrequest"`
}

type Base struct {
	Core Core  `nginx:"core"`
	HTTP *HTTP `nginx:"http"`
}

type KV struct {
	Key   string
	Value string
	// If this is true it means the value field was never set.
	HasValue bool
}

type User struct {
	Name  string
	Group string
}

type ThreadPool struct {
	Name     string
	Threads  int
	MaxQueue int
}

type ErrorLog struct {
	File  string
	Level string
}

type HTTP struct {
	SharedHTTP
	RequestPoolSize          int        `nginx:"request_pool_size"`
	Server                   ServerList `nginx:"server"`
	ServerNameHashBucketSize int        `nginx:"server_names_hash_bucket_size"`
	ServerNameHashMaxSize    int        `nginx:"server_names_hash_max_size"`
	UnderscoreInHeaders      bool       `nginx:"underscores_in_headers"`
	VariablesHashBucketSize  int        `nginx:"variables_hash_bucket_size"`
	VariablesHashMaxSize     int        `nginx:"variables_hash_max_size"`
}

type SharedHTTP struct {
	ErrorLog                 []ErrorLog        `nginx:"error_log"`
	AbsoluteRedirect         bool              `nginx:"absolute_redirect"`
	AIO                      string            `nginx:"aio"`
	AIOWrite                 bool              `nginx:"aio_write"`
	ChunkedTransferEncoding  bool              `nginx:"chunked_transfer_encoding"`
	ClientBodyBufferSize     int               `nginx:"client_body_buffer_size"`
	ClientBodyInFileOnly     bool              `nginx:"client_body_in_file_only"`
	ClientBodyInSingleBuffer bool              `nginx:"client_body_in_single_buffer"`
	ClientBodyTempPath       string            `nginx:"client_body_temp_path"`
	ClientBodyTimeout        time.Duration     `nginx:"client_body_timeout"`
	ClientHeaderBufferSize   int               `nginx:"client_header_buffer_size"`
	ClientHeaderTimeout      time.Duration     `nginx:"client_header_timeout"`
	ClientMaxBodySize        int               `nginx:"client_max_body_size"`
	ConnectionPoolSize       int               `nginx:"connection_pool_size"`
	DefaultType              string            `nginx:"default_type"`
	DirectIO                 string            `nginx:"directio"`
	DirectIOAlignment        int               `nginx:"directio_alignment"`
	DisableSymlink           *DisableSymlink   `nginx:"disable_symlinks"`
	ErrorPage                []ErrorPage       `nginx:"error_page"`
	Etag                     bool              `nginx:"etag"`
	IfModifiedSince          string            `nginx:"if_modified_since"`
	IgnoreInvalidHeaders     bool              `nginx:"ignore_invalid_headers"`
	DisableKeepAlive         string            `nginx:"keepalive_disable"`
	KeepAliveRequests        int               `nginx:"keepalive_requests"`
	KeepAliveTimeout         KeepAliveTimeout  `nginx:"keepalive_timeout"`
	LargeClientHeaderBuffers Buffer            `nginx:"large_client_header_buffers"`
	LimitRate                int               `nginx:"limit_rate"`
	LimitRateAfter           int               `nginx:"limit_rate_after"`
	LingeringClose           string            `nginx:"lingering_close"`
	LingeringTime            time.Duration     `nginx:"lingering_time"`
	LingeringTimeout         time.Duration     `nginx:"lingering_timeout"`
	LogNotFound              bool              `nginx:"log_not_found"`
	LogSubrequests           bool              `nginx:"log_subrequest"`
	MaxRanges                int               `nginx:"max_ranges"`
	MergeSlashes             int               `nginx:"merge_slashes"`
	MSEPadding               bool              `nginx:"msie_padding"`
	MSERefresh               bool              `nginx:"msie_refresh"`
	OpenFileCache            OpenFileCache     `nginx:"open_file_cache"`
	OpenFileCacheErrors      bool              `nginx:"open_file_cache_errors"`
	OpenFileCacheMinUses     int               `nginx:"open_file_cache_min_uses"`
	OpenFileCacheValid       time.Duration     `nginx:"open_file_cache_valid"`
	OutputBuffer             Buffer            `nginx:"output_buffers"`
	PortOnRedirect           bool              `nginx:"port_in_redirect"`
	PostponeOutput           int               `nginx:"postpone_output"`
	ReadAhead                int               `nginx:"read_ahead"`
	RecursiveErrorPage       bool              `nginx:"recursive_error_pages"`
	ResetTimeoutConnection   bool              `nginx:"reset_timedout_connection"`
	Resolver                 *DNSResolver      `nginx:"resolver"`
	ResolverTimeout          time.Duration     `nginx:"resolver_timeout"`
	Root                     string            `nginx:"root"`
	Satisfy                  string            `nginx:"satisfy"`
	SendIOWAT                int               `nginx:"send_lowat"`
	SendTimeout              time.Duration     `nginx:"send_timeout"`
	SendFile                 bool              `nginx:"sendfile"`
	SendFileMaxChunkSize     int               `nginx:"sendfile_max_chunk"`
	ServerNameInRedirect     bool              `nginx:"server_name_in_redirect"`
	ServerTokens             bool              `nginx:"server_tokens"`
	SubrequestBufferSize     int               `nginx:"subrequest_output_buffer_size"`
	TCPNoDelay               bool              `nginx:"tcp_nodelay"`
	TCPNoPush                bool              `nginx:"tcp_nopush"`
	Types                    map[string]string `nginx:"types"`
	TypesHashBucketSize      int               `nginx:"types_hash_bucket_size"`
	TypesHashMaxSize         int               `nginx:"types_hash_max_size"`
}

type DNSResolver struct {
	Address    []string
	Valid      time.Duration
	IPv6       bool
	StatusZone string
}

type OpenFileCache struct {
	Max      int
	Inactive time.Duration
	On       bool
}

type Buffer struct {
	Number int
	Size   int
}

type Mail struct {
	ErrorLog []ErrorLog `nginx:"error_log"`
}

type Stream struct {
	ErrorLog []ErrorLog `nginx:"error_log"`
}

type Server struct {
	SharedHTTP
	Listen              Listen      `nginx:"listen"`
	Location            []*Location `nginx:"location"`
	RequestPoolSize     int         `nginx:"request_pool_size"`
	ServerName          []string    `nginx:"server_name"`
	TryFiles            []string    `nginx:"try_files"`
	UnderscoreInHeaders bool        `nginx:"underscores_in_headers"`
}

type ServerFilterFunc func(*Server) bool

type ServerList []*Server

func (s ServerList) Filter(f ServerFilterFunc) ServerList {
	var ls ServerList
	for _, v := range s {
		if f(v) {
			ls = append(ls, v)
		}
	}
	return ls
}

type Location struct {
	SharedHTTP
	Location []*Location `nginx:"location"`
	TryFiles []string    `nginx:"try_files"`
}

type KeepAliveTimeout struct {
	Timeout time.Duration
	Header  time.Duration
}

type ErrorPage struct {
	Code []int
	Path string
}

type DisableSymlink struct {
	On      bool
	IfOwner bool
	From    string
}

type LimitAccept struct {
	Method string
	Allow  []string
	Deny   []string
}

type Listen struct {
	Address       *Connection
	SSL           bool
	HTTP2         bool
	SPDY          bool
	ProxyProtocol bool
	SetFib        int
	FastOpen      int
	Backlog       int
	RCVBuf        int
	SendBuf       int
	AcceptFilter  string
	Defered       bool
	Bind          bool
	IPv6Only      bool
	ReusePort     bool
	SoKeepalive   string
	Default       bool
}

type ConnType uint

const (
	Local ConnType = 1 + iota
	Remote
	Socket
)

// Connection represent various kinds of address that nginx accepts through
// configuration.
type Connection struct {
	Type      ConnType
	Localhost bool
	// This means we are listening to all interfaces. It can be defined by
	// *:80
	All  bool
	Port int
	IP   net.IP
	Net  *net.IPNet
	URL  *url.URL
}

func (c *Connection) String() string {
	switch c.Type {
	case Remote:
		// fully qualified url we are dealing with here. So either url or CIDR
		if c.URL != nil {
			return c.URL.String()
		}
		if c.Net != nil {
			return c.Net.String()
		}
		return ""
	case Socket:
		if c.URL != nil {
			return "unix:" + c.URL.Path
		}
		return ""
	case Local:
		var h string
		if c.Localhost {
			h = "localhost"
		}
		if c.IP != nil {
			h = c.IP.String()

		}
		if c.All {
			h = "*"
		}
		if c.Port != 0 {
			p := strconv.FormatInt(int64(c.Port), 10)
			if h != "" {
				h = net.JoinHostPort(h, p)
			} else {
				h = p
			}
		} else {
			if strings.IndexByte(h, ':') >= 0 {
				h = "[" + h + "]"
			}
		}
		return h
	default:
		return ""
	}
}
