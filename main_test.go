package traefik_responsebodyrewrite

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestServeHTTP(t *testing.T) {
	tests := []struct {
		desc            string
		httpStatus      int
		contentEncoding string
		responses       []Response
		lastModified    bool
		resBody         string
		expResBody      string
	}{
		{
			desc:       "should replace foo by bar",
			httpStatus: http.StatusOK,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
					},
				},
			},
			resBody:    "foo is the new bar",
			expResBody: "bar is the new bar",
		},
		{
			desc:       "should replace nothing",
			httpStatus: http.StatusNotFound,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
					},
				},
			},
			resBody:    "foo is the new bar",
			expResBody: "foo is the new bar",
		},
		{
			desc:       "should replace foo by bar, then by foo",
			httpStatus: http.StatusOK,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
						{
							Regex:       "bar",
							Replacement: "foo",
						},
					},
				},
			},
			resBody:    "foo is the new bar",
			expResBody: "foo is the new foo",
		},
		{
			desc:       "should not replace anything if content encoding is not identity or empty",
			httpStatus: http.StatusOK,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
					},
				},
			},
			contentEncoding: "gzip",
			resBody:         "foo is the new bar",
			expResBody:      "foo is the new bar",
		},
		{
			desc:       "should replace foo by bar if content encoding is identity",
			httpStatus: http.StatusOK,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
					},
				},
			},
			contentEncoding: "identity",
			resBody:         "foo is the new bar",
			expResBody:      "bar is the new bar",
		},
		{
			desc:       "should not remove the last modified header",
			httpStatus: http.StatusOK,
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
					},
				},
			},
			contentEncoding: "identity",
			lastModified:    true,
			resBody:         "foo is the new bar",
			expResBody:      "bar is the new bar",
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				Responses: test.responses,
			}

			next := func(rw http.ResponseWriter, req *http.Request) {
				rw.Header().Set("Content-Encoding", test.contentEncoding)
				rw.Header().Set("Last-Modified", "Thu, 02 Jun 2016 06:01:08 GMT")
				rw.Header().Set("Content-Length", strconv.Itoa(len(test.resBody)))
				rw.WriteHeader(test.httpStatus)

				_, _ = fmt.Fprintf(rw, test.resBody)
			}

			rewriteBody, err := New(context.Background(), http.HandlerFunc(next), config, "rewriteBody")
			if err != nil {
				t.Fatal(err)
			}

			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			rewriteBody.ServeHTTP(recorder, req)

			if _, exists := recorder.Result().Header["Content-Length"]; exists {
				t.Error("The Content-Length Header must be deleted")
			}

			if !bytes.Equal([]byte(test.expResBody), recorder.Body.Bytes()) {
				t.Errorf("got body %q, want %q", recorder.Body.Bytes(), test.expResBody)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		desc      string
		responses []Response
		expErr    bool
	}{
		{
			desc: "should return no error",
			responses: []Response{
				{
					Status: "200",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
						{
							Regex:       "bar",
							Replacement: "foo",
						},
					},
				},
			},
			expErr: false,
		},
		{
			desc: "should return no error",
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
						{
							Regex:       "bar",
							Replacement: "foo",
						},
					},
				},
			},
			expErr: false,
		},
		{
			desc: "should return no error",
			responses: []Response{
				{
					Status: "200-299,400",
					Rewrites: []Rewrite{
						{
							Regex:       "foo",
							Replacement: "bar",
						},
						{
							Regex:       "bar",
							Replacement: "foo",
						},
					},
				},
			},
			expErr: false,
		},
		{
			desc: "should return an error",
			responses: []Response{
				{
					Status: "200-299",
					Rewrites: []Rewrite{
						{
							Regex:       "*",
							Replacement: "bar",
						},
					},
				},
			},
			expErr: true,
		},
		{
			desc: "should return an error",
			responses: []Response{
				{
					Status: "200,299foo",
					Rewrites: []Rewrite{
						{
							Regex:       "*",
							Replacement: "bar",
						},
					},
				},
			},
			expErr: true,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			config := &Config{
				Responses: test.responses,
			}

			_, err := New(context.Background(), nil, config, "rewriteBody")
			if test.expErr && err == nil {
				t.Fatal("expected error on bad regexp format")
			}
		})
	}
}
