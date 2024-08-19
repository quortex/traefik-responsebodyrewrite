// Package responsebodyrewrite provides a middleware that rewrites the response body based on the status code and the content of the response.
package traefik_responsebodyrewrite

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"regexp"
)

// parsedRewrite holds one rewrite body configuration with parsed values.
type parsedRewrite struct {
	regex       *regexp.Regexp
	replacement []byte
}

// parsedResponse holds one response configuration with parsed values.
type parsedResponse struct {
	rewrites []parsedRewrite
	status   HTTPCodeRanges
}

// Rewrite holds one rewrite body configuration.
type Rewrite struct {
	Regex       string `json:"regex,omitempty"`
	Replacement string `json:"replacement,omitempty"`
}

// Response holds one response configuration.
type Response struct {
	Rewrites []Rewrite `json:"rewrites,omitempty"`
	Status   string    `json:"status,omitempty"`
}

// Config the plugin configuration.
type Config struct {
	Responses []Response `json:"responses,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		Responses: []Response{},
	}
}

// responsebodyrewrite is a middleware that rewrites the response body based on the status code and the content of the response.
type responsebodyrewrite struct {
	next       http.Handler
	name       string
	responses  []parsedResponse
	infoLogger *log.Logger
}

// New creates a new instance of the responsebodyrewrite middleware.
// It takes a context.Context, an http.Handler, a *Config, and a name string as parameters.
// It returns an http.Handler and an error.
func New(_ context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	infoLogger := log.New(io.Discard, "INFO: responsebodyrewrite: ", log.Ldate|log.Ltime)
	infoLogger.SetOutput(os.Stdout)
	infoLogger.Printf("Responses config: %s", config.Responses)

	parsedResponses := make([]parsedResponse, len(config.Responses))
	for i, response := range config.Responses {
		// Parse the HTTP code ranges
		httpCodeRanges, err := NewHTTPCodeRanges([]string{response.Status})
		if err != nil {
			return nil, err
		}

		// Parse the rewrites
		rewrites := make([]parsedRewrite, len(response.Rewrites))
		for i, rewriteConfig := range response.Rewrites {
			regex, err := regexp.Compile(rewriteConfig.Regex)
			if err != nil {
				return nil, fmt.Errorf("error compiling regex %q: %w", rewriteConfig.Regex, err)
			}

			rewrites[i] = parsedRewrite{
				regex:       regex,
				replacement: []byte(rewriteConfig.Replacement),
			}
		}

		parsedResponses[i] = parsedResponse{
			rewrites: rewrites,
			status:   httpCodeRanges,
		}
	}

	return &responsebodyrewrite{
		responses:  parsedResponses,
		next:       next,
		name:       name,
		infoLogger: infoLogger,
	}, nil
}

// ServeHTTP is the method that handles the HTTP request.
// It rewrites the response body based on the status code and the content of the response.
func (r *responsebodyrewrite) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	wrappedWriter := &responseWriter{
		code:           http.StatusOK,
		headerMap:      make(http.Header),
		ResponseWriter: rw,
		responses:      r.responses,
	}

	r.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.buffer.Bytes()

	for _, response := range r.responses {
		if !response.status.Contains(wrappedWriter.code) {
			continue
		}
		for _, rewrite := range response.rewrites {
			bodyBytes = rewrite.regex.ReplaceAll(bodyBytes, rewrite.replacement)
		}
		break
	}

	if _, err := rw.Write(bodyBytes); err != nil {
		r.infoLogger.Printf("unable to write body: %v", err)
	}

}

// responseWriter is a wrapper around an http.ResponseWriter that allows us to intercept the response.
// It implements the http.ResponseWriter interface.
type responseWriter struct {
	buffer      bytes.Buffer
	headerMap   http.Header
	headersSent bool
	code        int
	http.ResponseWriter
	responses []parsedResponse
}

// Headers implements the http.ResponseWriter interface.
func (rw *responseWriter) Header() http.Header {
	if rw.headersSent {
		return rw.ResponseWriter.Header()
	}

	if rw.headerMap == nil {
		rw.headerMap = make(http.Header)
	}

	return rw.headerMap
}

// WriteHeader implements the http.ResponseWriter interface.
// It intercepts the response status code and stores it in the responseWriter struct.
func (rw *responseWriter) WriteHeader(statusCode int) {
	if rw.headersSent {
		return
	}

	// Handling informational headers.
	if statusCode >= 100 && statusCode <= 199 {
		// Multiple informational status codes can be used,
		// so here the copy is not appending the values to not repeat them.
		for k, v := range rw.Header() {
			rw.ResponseWriter.Header()[k] = v
		}

		rw.ResponseWriter.WriteHeader(statusCode)
		return
	}

	rw.code = statusCode

	// Check if the status code is in the list of status codes to rewrite.
	var bodyRewrited bool
	for _, response := range rw.responses {
		if !response.status.Contains(statusCode) {
			continue
		}
		bodyRewrited = true
	}

	// Remove Content-Length header if the body is being rewritten.
	if bodyRewrited {
		rw.ResponseWriter.Header().Del("Content-Length")
	}

	rw.ResponseWriter.WriteHeader(rw.code)
	rw.headersSent = true
}

// Write implements the http.ResponseWriter interface.
func (rw *responseWriter) Write(p []byte) (int, error) {

	if !rw.headersSent {
		rw.WriteHeader(http.StatusOK)
	}

	return rw.buffer.Write(p)
}

// Hijack implements the http.Hijacker interface.
func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}

	return nil, nil, fmt.Errorf("not a hijacker: %T", rw.ResponseWriter)
}

// Flush implements the http.Flusher interface.
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
