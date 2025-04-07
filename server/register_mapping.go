package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/default23/protofake/mapper"
)

// SetMappings replaces the current mappings with the provided ones.
func (s *Server) SetMappings(mappings []*mapper.Mapping) error {
	endpointMappings := make(map[string][]*mapper.Mapping)
	for _, m := range mappings {
		if err := s.isMappingApplicable(m); err != nil {
			return fmt.Errorf("the mapping (id=%s enpoint=%s) is invalid: %w", m.ID, m.Endpoint, err)
		}

		endpointMappings[m.Endpoint] = append(endpointMappings[m.Endpoint], m)
	}
	for k := range endpointMappings {
		if len(endpointMappings[k]) == 0 {
			delete(endpointMappings, k)
			continue
		}

		slog.Debug("registered endpoint mappings", "endpoint", k, "mappings_count", len(endpointMappings[k]))
	}

	s.mappings = endpointMappings
	return nil
}

func (s *Server) isMappingApplicable(m *mapper.Mapping) error {
	if err := m.IsValid(); err != nil {
		return fmt.Errorf("invalid mapping '%s': %w", m.Endpoint, err)
	}

	endpoint := strings.Trim(m.Endpoint, "/")
	endpointParts := strings.Split(endpoint, "/")
	serviceName := endpointParts[0]
	methodName := endpointParts[1]

	fullMethodName := fmt.Sprintf("/%s/%s", serviceName, methodName)

	serviceDesc, ok := s.services[serviceName]
	if !ok {
		return fmt.Errorf("endpoint '%s' provided in mapping are not registered", m.Endpoint)
	}

	var found bool
	for _, method := range serviceDesc.Service.Methods {
		if method.MethodName == methodName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("method '%s' not implemented by service '%s'", methodName, serviceName)
	}

	mf, ok := s.messageFactory[fullMethodName]
	if !ok {
		return fmt.Errorf("internal server error: message could not be constructed for this endpoint '%s'", fullMethodName)
	}

	in, out := mf()

	_, inJsonBytes, err := marshalProtoMessage(in)
	if err != nil {
		return fmt.Errorf("internal error: processing input message for method %q: %v", fullMethodName, err)
	}

	_, outJsonBytes, err := marshalProtoMessage(out)
	if err != nil {
		return fmt.Errorf("internal error: processing input message for method %q: %v", fullMethodName, err)
	}

	for valuePath := range m.RequestBody {
		ej := gjson.GetBytes(inJsonBytes, valuePath)
		if !ej.Exists() {
			return fmt.Errorf("the json path %q is provided, but not exists in INPUT message", valuePath)
		}

		// TODO check for the value types of request and provided value matcher
	}
	for valuePath := range m.Response.Body {
		j := gjson.GetBytes(outJsonBytes, valuePath)
		if !j.Exists() {
			return fmt.Errorf("the json path %q is provided, but not exists in OUTPUT message", valuePath)
		}

		// TODO check for the value types of response and provided response mapper
	}

	return nil
}

func marshalProtoMessage(pm interface {
	Interface() protoreflect.ProtoMessage
}) (map[string]any, []byte, error) {
	out := make(map[string]any)
	if pm == nil { // emptypb.Empty for example
		return out, []byte("{}"), nil
	}

	message := pm.Interface()

	mo := protojson.MarshalOptions{
		UseProtoNames:     true,
		EmitUnpopulated:   true,
		EmitDefaultValues: true,
	}
	outBytes, err := mo.Marshal(message)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to verify output message (convert to json bytes): %w", err)
	}
	if err = json.Unmarshal(outBytes, &out); err != nil {
		return nil, nil, fmt.Errorf("failed to verify output message(convert bytes to object): %w", err)
	}

	return out, outBytes, nil
}
