package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
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

	for valuePath, matcher := range m.RequestBody {
		ej := gjson.GetBytes(inJsonBytes, valuePath)
		if !ej.Exists() {
			return fmt.Errorf("the json path %q is provided, but not exists in INPUT message", valuePath)
		}

		// TODO: it may not be working on other rules
		matcherValueType := reflect.TypeOf(matcher.Value)
		messageValueType := reflect.TypeOf(ej.Value())
		if matcherValueType != messageValueType {
			return fmt.Errorf("the json path %q is provided, but the value type is not equal to the expected type, should be of type: %s, got: %s", valuePath, messageValueType, matcherValueType)
		}
	}

	for valuePath, value := range m.Response.Body {
		j := gjson.GetBytes(outJsonBytes, valuePath)
		if !j.Exists() {
			return fmt.Errorf("the json path %q is provided, but not exists in OUTPUT message", valuePath)
		}

		valueType := reflect.TypeOf(value)
		if valueType.Kind() == reflect.String && strings.HasPrefix(value.(string), "$") {
			// TODO check the given path ($req.body.some_name) exists in request body
			continue
		}

		messageValueType := reflect.TypeOf(j.Value())
		if valueType != messageValueType {
			return fmt.Errorf("the json path %q is provided, but the value type is not equal to the expected type, should be of type: %s, got: %s", valuePath, messageValueType, valueType)
		}

		// TODO check for the value types of response and provided response mapper
		// TODO response property could be a slice of objects, check this out
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

	message := proto.Clone(pm.Interface()) // to avoid modifying the original message
	initDefaults(message.ProtoReflect())

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

func initDefaults(m protoreflect.Message) {
	md := m.Descriptor()
	for i := 0; i < md.Fields().Len(); i++ {
		fd := md.Fields().Get(i)

		// is for "optional" primitive fields, which are have nil by default.
		if !m.Has(fd) {
			newVal := m.NewField(fd)
			m.Set(fd, newVal)
		}

		// is for message fields, which are have nil by default.
		if fd.Kind() == protoreflect.MessageKind && !fd.IsList() && !fd.IsMap() {
			if !m.Has(fd) {
				newMsg := m.NewField(fd)
				m.Set(fd, newMsg)

				initDefaults(newMsg.Message())
			} else {
				childMsg := m.Get(fd).Message()
				initDefaults(childMsg)
			}
		}
	}
}
