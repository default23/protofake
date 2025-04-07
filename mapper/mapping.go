package mapper

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
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
	ID          string                  `json:"id"`
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

// IsValid checks if the mapping is valid, if not it returns an error.
// MUTATES the mapping with the default values.
func (m *Mapping) IsValid() error {
	if m.Endpoint == "" {
		return fmt.Errorf("mapping '%s' does not contain an endpoint", m.Endpoint)
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
	}
	if m.Response.Body == nil {
		m.Response.Body = make(map[string]any)
	}
	if m.Response.Code == "" {
		m.Response.Code = "OK"
	}

	endpointParts := strings.Split(strings.Trim(m.Endpoint, "/"), "/")
	if len(endpointParts) != 2 {
		return fmt.Errorf("mapping endpoint '%s' should be in the format 'package.service/method'", m.Endpoint)
	}

	for k, v := range m.Metadata {
		if _, err := NewValueMatcher(v.Rule, v.Value); err != nil {
			return fmt.Errorf("invalid value matcher for key '%s' in mapping '%s': %w", k, m.Endpoint, err)
		}
	}
	for k, v := range m.RequestBody {
		if _, err := NewValueMatcher(v.Rule, v.Value); err != nil {
			return fmt.Errorf("invalid value matcher for key '%s' in mapping '%s': %w", k, m.Endpoint, err)
		}
	}

	if _, ok := StrToCode[m.Response.Code]; !ok {
		return fmt.Errorf("invalid response code '%s' in mapping with id=%s, endpoint: '%s'", m.Response.Code, m.ID, m.Endpoint)
	}

	return nil
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
