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
	next         http.Handler
	name         string
	responses    []parsedResponse
	lastModified bool
	infoLogger   *log.Logger
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
		responses:    parsedResponses,
		next:         next,
		name:         name,
		lastModified: true,
		infoLogger:   infoLogger,
	}, nil
}

// ServeHTTP is the method that handles the HTTP request.
// It rewrites the response body based on the status code and the content of the response.
func (r *responsebodyrewrite) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	wrappedWriter := &responseWriter{
		lastModified:   r.lastModified,
		code:           http.StatusOK,
		headerMap:      make(http.Header),
		ResponseWriter: rw,
	}

	r.next.ServeHTTP(wrappedWriter, req)

	bodyBytes := wrappedWriter.buffer.Bytes()

	contentEncoding := wrappedWriter.Header().Get("Content-Encoding")

	if contentEncoding != "" && contentEncoding != "identity" {
		if _, err := rw.Write(bodyBytes); err != nil {
			r.infoLogger.Printf("unable to write body: %v", err)
		}

		return
	}

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
		r.infoLogger.Printf("unable to write rewrited body: %v", err)
	}

}

// responseWriter is a wrapper around an http.ResponseWriter that allows us to intercept the response.
// It implements the http.ResponseWriter interface.
type responseWriter struct {
	buffer       bytes.Buffer
	lastModified bool
	headerMap    http.Header
	headersSent  bool
	code         int
	http.ResponseWriter
}

// Headers implements the http.ResponseWriter interface.
func (r *responseWriter) Header() http.Header {
	if r.headersSent {
		return r.ResponseWriter.Header()
	}

	if r.headerMap == nil {
		r.headerMap = make(http.Header)
	}

	return r.headerMap
}

// WriteHeader implements the http.ResponseWriter interface.
func (r *responseWriter) WriteHeader(statusCode int) {
	if r.headersSent {
		return
	}

	// Handling informational headers.
	if statusCode >= 100 && statusCode <= 199 {
		// Multiple informational status codes can be used,
		// so here the copy is not appending the values to not repeat them.
		for k, v := range r.Header() {
			r.ResponseWriter.Header()[k] = v
		}

		r.ResponseWriter.WriteHeader(statusCode)
		return
	}

	r.code = statusCode

	// The copy is not appending the values,
	// to not repeat them in case any informational status code has been written.
	for k, v := range r.Header() {
		r.ResponseWriter.Header()[k] = v
	}

	// Delegates the Content-Length Header creation to the final body write.
	r.ResponseWriter.Header().Del("Content-Length")
	r.ResponseWriter.WriteHeader(r.code)
	r.headersSent = true
}

// Write implements the http.ResponseWriter interface.
func (r *responseWriter) Write(p []byte) (int, error) {

	if !r.headersSent {
		r.WriteHeader(http.StatusOK)
	}

	return r.buffer.Write(p)
}

// Hijack implements the http.Hijacker interface.
func (r *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, fmt.Errorf("%T is not a http.Hijacker", r.ResponseWriter)
	}

	return hijacker.Hijack()
}

// Flush implements the http.Flusher interface.
func (r *responseWriter) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
