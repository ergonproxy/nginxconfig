package helpers

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ergongate/nginxconfig/config"
)

// regexp for matching url
const (
	URLPort = `(:(\d{1,5})$)`
	Numeric = "^[0-9]+$"
)

var regURLPORT = regexp.MustCompile(URLPort)
var regIsNumber = regexp.MustCompile(Numeric)

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
	if regURLPORT.MatchString(txt) {
		if strings.HasPrefix(txt, "*:") {
			pn, err := strconv.Atoi(txt[2:])
			if err != nil {
				return nil, err
			}
			return &config.Connection{Type: config.Local, All: true, Port: pn}, nil
		}
		host, port, err := net.SplitHostPort(txt)
		if err != nil {
			return nil, err
		}
		pn, err := strconv.Atoi(port)
		if err != nil {
			return nil, err
		}
		c := &config.Connection{Type: config.Local, Port: pn}
		if host == "localhost" {
			c.Localhost = true
		}
		ip := net.ParseIP(host)
		if ip != nil {
			c.IP = ip
			return c, nil
		}
		u, err := url.Parse(txt)
		if err != nil {
			return nil, ErrUknownConnection
		}
		c.Type = config.Remote
		if u.Scheme == "unix" {
			c.Type = config.Socket
		} else {
			c.Type = config.Remote
		}
		c.URL = u
		return c, nil
	}
	if regIsNumber.MatchString(txt) {
		// we only have a port
		pn, err := strconv.Atoi(txt)
		if err != nil {
			return nil, err
		}
		return &config.Connection{Type: config.Local, Port: pn}, nil
	}
	if txt[0] == '[' && txt[len(txt)-1] == ']' {
		//ipv6
		txt = txt[1 : len(txt)-1]
	}
	ip := net.ParseIP(txt)
	if ip != nil {
		return &config.Connection{Type: config.Local, IP: ip}, nil
	}
	c := &config.Connection{}
	u, err := url.Parse(txt)
	if err != nil {
		fmt.Println(err)
		return nil, ErrUknownConnection
	}
	c.Type = config.Remote
	if u.Scheme == "unix" {
		c.Type = config.Socket
	} else {
		c.Type = config.Remote
	}
	c.URL = u
	return c, nil
}

func IsPort(str string) bool {
	if i, err := strconv.Atoi(str); err == nil && i > 0 && i < 65536 {
		return true
	}
	return false
}
