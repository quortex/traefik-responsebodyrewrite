package traefik_responsebodyrewrite

import (
	"reflect"
	"testing"
)

func TestHTTPCodeRanges_Contains(t *testing.T) {
	tests := []struct {
		desc        string
		ranges      HTTPCodeRanges
		statusCode  int
		expectedRes bool
	}{
		{
			desc:        "should return true if status code is within the range",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  200,
			expectedRes: true,
		},
		{
			desc:        "should return true if status code is at the lower bound of the range",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  400,
			expectedRes: true,
		},
		{
			desc:        "should return true if status code is at the upper bound of the range",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  499,
			expectedRes: true,
		},
		{
			desc:        "should return false if status code is outside the range",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  300,
			expectedRes: false,
		},
		{
			desc:        "should return false if status code is below all ranges",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  100,
			expectedRes: false,
		},
		{
			desc:        "should return false if status code is above all ranges",
			ranges:      HTTPCodeRanges{{200, 299}, {400, 499}},
			statusCode:  500,
			expectedRes: false,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res := test.ranges.Contains(test.statusCode)
			if res != test.expectedRes {
				t.Errorf("got %v, want %v", res, test.expectedRes)
			}
		})
	}
}
func TestNewHTTPCodeRanges(t *testing.T) {
	tests := []struct {
		desc      string
		strBlocks []string
		expected  HTTPCodeRanges
		expectErr bool
	}{
		{
			desc:      "should create HTTPCodeRanges with single code",
			strBlocks: []string{"200"},
			expected:  HTTPCodeRanges{{200, 200}},
			expectErr: false,
		},
		{
			desc:      "should create HTTPCodeRanges with code range",
			strBlocks: []string{"200-299"},
			expected:  HTTPCodeRanges{{200, 299}},
			expectErr: false,
		},
		{
			desc:      "should create HTTPCodeRanges with multiple code ranges",
			strBlocks: []string{"200-299", "400-499"},
			expected:  HTTPCodeRanges{{200, 299}, {400, 499}},
			expectErr: false,
		},
		{
			desc:      "should create HTTPCodeRanges with multiple code ranges",
			strBlocks: []string{"200-299", "400", "450-499"},
			expected:  HTTPCodeRanges{{200, 299}, {400, 400}, {450, 499}},
			expectErr: false,
		},
		{
			desc:      "should return error for invalid code",
			strBlocks: []string{"200-abc"},
			expected:  nil,
			expectErr: true,
		},
		{
			desc:      "should return error for invalid code",
			strBlocks: []string{"abc-200"},
			expected:  nil,
			expectErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			res, err := NewHTTPCodeRanges(test.strBlocks)
			if test.expectErr && err == nil {
				t.Errorf("expected error, but got nil")
			}
			if !test.expectErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, test.expected) {
				t.Errorf("got %v, want %v", res, test.expected)
			}
		})
	}
}
