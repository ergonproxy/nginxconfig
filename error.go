package main

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// source https://en.wikipedia.org/wiki/List_of_HTTP_status_codes#nginx
const (
	// Used internally[89] to instruct the server to return no information to the
	// client and close the connection immediately.
	statusNoResponse = 444
	// Client sent too large request or too long header line.
	statusRequestHeaderTooLarge = 494
	// An expansion of the 400 Bad Request response code, used when the client has
	// provided an invalid client certificate.
	statusSSLCertificateError = 495
	// An expansion of the 400 Bad Request response code, used when a client
	// certificate is required but not provided.
	statusSSLCertificateRequired = 496
	// An expansion of the 400 Bad Request response code, used when the client has
	// made a HTTP request to a port listening for HTTPS requests.
	statusHTTPToHTTPSPort = 497
	// Used when the client has closed the request before the server could send a
	// response.
	statusClientClosedRequest = 499
)

const docsSite = "https://docs.vince.co.tz/"

var statusTextMap = map[int]string{
	statusNoResponse:             "No Response",
	statusRequestHeaderTooLarge:  "Request header too large",
	statusSSLCertificateError:    "SSL Certificate Error",
	statusSSLCertificateRequired: "SSL Certificate Required",
	statusHTTPToHTTPSPort:        "HTTP Request Sent to HTTPS Port",
	statusClientClosedRequest:    "Client Closed Request",
}

var statusCodesLock sync.Mutex

func statusText(code int) string {
	if code >= 444 && code <= 499 {
		statusCodesLock.Lock()
		txt, ok := statusTextMap[code]
		statusCodesLock.Unlock()
		if ok {
			return txt
		}
	}
	return http.StatusText(code)
}

type httpError struct {
	Status int    `json:"status"`
	Text   string `json:"text"`
	Code   string `json:"code"`
}

type httpErrorResponse struct {
	Error     httpError `json:"error"`
	RequestID string    `json:"request_id"`
	HREF      string    `json:"href"`
}

func newHTTPErrorResponse(code int, id string) httpErrorResponse {
	return httpErrorResponse{
		Error: httpError{
			Status: code,
			Text:   statusText(code),
		},
		RequestID: id,
		HREF:      linkErrorDocs(code),
	}
}

func linkDocs(page ...string) string {
	return docsSite + strings.Join(page, "/")
}

func linkErrorDocs(code int) string {
	return linkDocs("errors", strconv.FormatInt(int64(code), 10))
}
