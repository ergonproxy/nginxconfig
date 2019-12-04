package helpers

import (
	"errors"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ergongate/nginxconfig/config"
)

// ErrInvalidFlag  is returned for wrong flag values.
var ErrInvalidFlag = errors.New("invalid flag")
var ErrUknownConnection = errors.New("unknown connection")

// ParseFlag checks whether txt is a nginx flag value. Flag value can either be
// on or off, this maps on to true and off to false.
func ParseFlag(txt string) (bool, error) {
	n := strings.TrimSpace(txt)
	n = strings.ToLower(n)
	switch n {
	case "on":
		return true, nil
	case "off":
		return false, nil
	default:
		return false, ErrInvalidFlag
	}
}

// ParseDuration parses nginx time values to time.Duration
func ParseDuration(txt string) (time.Duration, error) {
	txt = strings.TrimSpace(txt)
	return time.ParseDuration(txt)
}

// ParseConnection parses common connection strings expected by nginx
// address | CIDR | unix:
func ParseConnection(txt string) (*config.Connection, error) {
	for _, mode := range []config.ConnType{config.Address, config.IP, config.CIDR} {
		switch mode {
		case config.IP:
			ip := net.ParseIP(txt)
			if ip != nil {
				return &config.Connection{Type: mode, IP: ip}, nil
			}
		case config.Address:
			u, err := url.Parse(txt)
			if err == nil {
				return &config.Connection{Type: mode, URL: u}, nil
			}
		case config.CIDR:
			ip, addr, err := net.ParseCIDR(txt)
			if err == nil {
				return &config.Connection{Type: mode, IP: ip, Net: addr}, nil
			}
		}
	}
	return nil, ErrUknownConnection
}
