package endpoint

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewEndpoint(t *testing.T) {
	type args struct {
		scheme string
		host   string
	}
	tests := []struct {
		name string
		args args
		want *url.URL
	}{
		{
			name: "https://github.com/webitel/webitel-wfm/",
			args: args{"https", "github.com/webitel/webitel-wfm/"},
			want: &url.URL{Scheme: "https", Host: "github.com/webitel/webitel-wfm/"},
		},
		{
			name: "https://webitel.com/",
			args: args{"https", "webitel.com/"},
			want: &url.URL{Scheme: "https", Host: "webitel.com/"},
		},
		{
			name: "https://www.google.com/",
			args: args{"https", "www.google.com/"},
			want: &url.URL{Scheme: "https", Host: "www.google.com/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, NewEndpoint(tt.args.scheme, tt.args.host), tt.want)
		})
	}
}

func TestParseEndpoint(t *testing.T) {
	type args struct {
		endpoints []string
		scheme    string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "webitel",
			args:    args{endpoints: []string{"https://github.com/webitel/webitel-wfm"}, scheme: "https"},
			want:    "github.com",
			wantErr: false,
		},
		{
			name:    "test",
			args:    args{endpoints: []string{"http://webitel.com/"}, scheme: "https"},
			want:    "",
			wantErr: false,
		},
		{
			name:    "localhost:8080",
			args:    args{endpoints: []string{"grpcs://localhost:8080/"}, scheme: "grpcs"},
			want:    "localhost:8080",
			wantErr: false,
		},
		{
			name:    "localhost:8081",
			args:    args{endpoints: []string{"grpcs://localhost:8080/"}, scheme: "grpc"},
			want:    "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseEndpoint(tt.args.endpoints, tt.args.scheme)
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSchema(t *testing.T) {
	tests := []struct {
		schema string
		secure bool
		want   string
	}{
		{
			schema: "http",
			secure: true,
			want:   "https",
		},
		{
			schema: "http",
			secure: false,
			want:   "http",
		},
		{
			schema: "grpc",
			secure: true,
			want:   "grpcs",
		},
		{
			schema: "grpc",
			secure: false,
			want:   "grpc",
		},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, Scheme(tt.schema, tt.secure))
	}
}
