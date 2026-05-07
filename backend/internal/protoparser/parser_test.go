package protoparser

import (
	"errors"
	"testing"
)

func TestParseProjectProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		src        string
		wantErr    error
		wantSvc    string
		wantMethod string
		wantParam  string
	}{
		{
			name: "success",
			src: `
syntax = "proto3";

service UserService {
  rpc GetUser (GetUserRequest) returns (GetUserResponse);
}

message GetUserRequest {
  string user_id = 1;
  int64 account_id = 2;
}
`,
			wantSvc:    "UserService",
			wantMethod: "GetUser",
			wantParam:  "user_id",
		},
		{
			name: "with comments and stream",
			src: `
// comment
service EventService {
  rpc StreamEvents (stream EventsRequest) returns (stream EventsResponse); // trailing comment
}
message EventsRequest {
  string app = 1;
}
`,
			wantSvc:    "EventService",
			wantMethod: "StreamEvents",
			wantParam:  "app",
		},
		{
			name:    "no service",
			src:     `syntax = "proto3"; message A { string id = 1; }`,
			wantErr: ErrInvalidProto,
		},
		{
			name:    "service without rpc",
			src:     `service A {}`,
			wantErr: ErrInvalidProto,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := ParseProjectProto([]byte(tt.src))
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v", err)
			}
			if got.ServiceName != tt.wantSvc {
				t.Fatalf("service = %q, want %q", got.ServiceName, tt.wantSvc)
			}
			if len(got.Methods) == 0 || got.Methods[0].Name != tt.wantMethod {
				t.Fatalf("first method = %q, want %q", firstMethod(got), tt.wantMethod)
			}
			if len(got.Methods[0].Parameters) == 0 || got.Methods[0].Parameters[0].Name != tt.wantParam {
				t.Fatalf("first param = %q, want %q", firstParam(got), tt.wantParam)
			}
		})
	}
}

func firstMethod(c Contract) string {
	if len(c.Methods) == 0 {
		return ""
	}
	return c.Methods[0].Name
}

func firstParam(c Contract) string {
	if len(c.Methods) == 0 || len(c.Methods[0].Parameters) == 0 {
		return ""
	}
	return c.Methods[0].Parameters[0].Name
}

