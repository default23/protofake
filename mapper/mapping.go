package mapper

import (
	"encoding/json"
	"strings"

	"github.com/tidwall/gjson"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

var StrToCode = map[string]codes.Code{
	`OK`:                  codes.OK,
	`CANCELLED`:           codes.Canceled, //nolint
	`UNKNOWN`:             codes.Unknown,
	`INVALID_ARGUMENT`:    codes.InvalidArgument,
	`DEADLINE_EXCEEDED`:   codes.DeadlineExceeded,
	`NOT_FOUND`:           codes.NotFound,
	`ALREADY_EXISTS`:      codes.AlreadyExists,
	`PERMISSION_DENIED`:   codes.PermissionDenied,
	`RESOURCE_EXHAUSTED`:  codes.ResourceExhausted,
	`FAILED_PRECONDITION`: codes.FailedPrecondition,
	`ABORTED`:             codes.Aborted,
	`OUT_OF_RANGE`:        codes.OutOfRange,
	`UNIMPLEMENTED`:       codes.Unimplemented,
	`INTERNAL`:            codes.Internal,
	`UNAVAILABLE`:         codes.Unavailable,
	`DATA_LOSS`:           codes.DataLoss,
	`UNAUTHENTICATED`:     codes.Unauthenticated,
}

// Mapping verifies the incoming message is satisfying the defined rules.
type Mapping struct {
	Endpoint    string                  `json:"endpoint"`
	Metadata    map[string]ValueMatcher `json:"metadata"`
	RequestBody map[string]ValueMatcher `json:"request_body"`
	Response    Response                `json:"response"`
}

// Response is the output values.
type Response struct {
	Code string         `json:"code"`
	Body map[string]any `json:"body"`
	// ErrorMessage is applied when the Code is not codes.OK.
	ErrorMessage string `json:"error_message"`
}

// Matches checks if the given request can be processed by Mapping.
func (m *Mapping) Matches(md metadata.MD, body map[string]any) bool {
	if !m.matchesMetadata(md) {
		return false
	}

	return match(body, m.RequestBody)
}

func (m *Mapping) matchesMetadata(md metadata.MD) bool {
	mdobj := make(map[string]any, md.Len())
	for k, v := range md {
		mdobj[k] = strings.Join(v, ",")
	}

	return match(mdobj, m.Metadata)
}

func match(target map[string]any, mappings map[string]ValueMatcher) bool {
	jsonBody, _ := json.Marshal(target)

	for key, matcher := range mappings {
		data := gjson.GetBytes(jsonBody, key)
		if !data.Exists() {
			return false
		}

		if !matcher.Matches(data.Value()) {
			return false
		}
	}

	return true
}
