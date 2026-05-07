package githubclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type testDoer struct {
	resp *http.Response
	err  error
	req  *http.Request
}

func (d *testDoer) Do(req *http.Request) (*http.Response, error) {
	d.req = req
	if d.err != nil {
		return nil, d.err
	}
	return d.resp, nil
}

func TestReadProjectProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		repoURL       string
		ref           string
		token         string
		respStatus    int
		respBody      string
		doErr         error
		wantErr       error
		wantContains  string
		wantAuth      string
		wantQueryPart string
	}{
		{
			name:          "success with ref",
			repoURL:       "https://github.com/acme/service-a",
			ref:           "main",
			token:         "tkn",
			respStatus:    http.StatusOK,
			respBody:      `{"content":"c3ludGF4ID0gInByb3RvMyI7","encoding":"base64"}`,
			wantContains:  `syntax = "proto3";`,
			wantAuth:      "Bearer tkn",
			wantQueryPart: "ref=main",
		},
		{
			name:    "not found",
			repoURL: "https://github.com/acme/service-a",
			respStatus: http.StatusNotFound,
			respBody: "{}",
			wantErr: ErrProtoNotFound,
		},
		{
			name:    "invalid repo url",
			repoURL: "https://gitlab.com/acme/service-a",
			wantErr: ErrInvalidRepositoryURL,
		},
		{
			name:    "transport error",
			repoURL: "https://github.com/acme/service-a",
			doErr:   errors.New("network"),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			doer := &testDoer{err: tt.doErr}
			if tt.respStatus != 0 {
				doer.resp = &http.Response{
					StatusCode: tt.respStatus,
					Body:       io.NopCloser(strings.NewReader(tt.respBody)),
				}
			}
			c := New(doer, tt.token)

			got, err := c.ReadProjectProto(context.Background(), tt.repoURL, tt.ref)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if tt.doErr != nil {
				if err == nil || !strings.Contains(err.Error(), tt.doErr.Error()) {
					t.Fatalf("error = %v, want contains %q", err, tt.doErr.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if !strings.Contains(string(got), tt.wantContains) {
				t.Fatalf("proto = %q, want contains %q", string(got), tt.wantContains)
			}
			if tt.wantAuth != "" && doer.req.Header.Get("Authorization") != tt.wantAuth {
				t.Fatalf("authorization = %q, want %q", doer.req.Header.Get("Authorization"), tt.wantAuth)
			}
			if tt.wantQueryPart != "" && !strings.Contains(doer.req.URL.RawQuery, tt.wantQueryPart) {
				t.Fatalf("query = %q, want contains %q", doer.req.URL.RawQuery, tt.wantQueryPart)
			}
		})
	}
}

