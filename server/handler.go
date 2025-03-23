package server

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"github.com/tidwall/sjson"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/default23/protofake/mapper"
)

var emptyMessage = &emptypb.Empty{}

// NewMockHandler is the constructor for the gRPC method handler.
// Handles any incoming request for registered gRPC methods and applies the configured mappings on it.
func (s *Server) NewMockHandler(
	protoDescr *descriptorpb.FileDescriptorProto,
	serviceDescr *descriptorpb.ServiceDescriptorProto,
	methodDescr *descriptorpb.MethodDescriptorProto,
) (grpc.MethodHandler, error) {
	fullMethodName := fmt.Sprintf("/%s.%s/%s", protoDescr.GetPackage(), serviceDescr.GetName(), methodDescr.GetName())

	msgFactory, err := NewMessageFactory(protoDescr, methodDescr)
	if err != nil {
		return nil, fmt.Errorf("construct the messages factory for method %s: %w", fullMethodName, err)
	}
	s.messageFactory[fullMethodName] = msgFactory

	return func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in gRPC handler", "error", r)
			}
		}()

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			md = metadata.MD{}
		}

		logger := slog.With(
			"method", fullMethodName,
			"input_type", methodDescr.GetInputType(),
			"output_type", methodDescr.GetOutputType(),
			"metadata", md,
		)

		in, out := msgFactory()
		if err = dec(in); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode input message: %v", err))
		}

		msgIn := parseMessage(in)

		mappings := s.mappings[fullMethodName]
		if len(mappings) == 0 {
			logger.Warn("no mappings registered for method")
			return nil, status.Error(codes.FailedPrecondition, "no mappings registered for method "+fullMethodName)
		}

		var shouldRespondWith *mapper.Response
		// iterate backwards over the mappings
		// because the last mapping is the most recent added mapping.
		for _, mapping := range slices.Backward(mappings) {
			if !mapping.Matches(md, msgIn) {
				continue
			}

			shouldRespondWith = &mapping.Response
			break
		}
		if shouldRespondWith == nil {
			logger.Warn("no matching response mapping found")
			return nil, status.Error(codes.FailedPrecondition, "no one of registered mappings matches")
		}

		responseCode := mapper.StrToCode[shouldRespondWith.Code]

		if shouldRespondWith.ErrorMessage != "" || responseCode != codes.OK {
			logger.Debug("returning error response", "code", shouldRespondWith.Code, "error", shouldRespondWith.ErrorMessage)
			return nil, status.Error(responseCode, shouldRespondWith.ErrorMessage)
		}
		if out == nil {
			logger.Debug("returning empty response, because the method output message is not supposed to be used")
			return emptyMessage, nil
		}

		outjson := "{}"
		for k, v := range shouldRespondWith.Body {
			outjson, err = sjson.Set(outjson, k, v)
			if err != nil {
				return nil, status.Error(codes.FailedPrecondition, "failed to set property "+k+" in output message, verify the registered mappings: "+err.Error())
			}
		}

		unmarshalOpts := protojson.UnmarshalOptions{
			DiscardUnknown: s.config.DiscardUnknownFields,
		}
		if err = unmarshalOpts.Unmarshal([]byte(outjson), out.Interface()); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "check registered mappings for method, failed to unmarshal output message into %s message: %v", methodDescr.GetOutputType(), err.Error())
		}

		logger.Debug("successfully mapped gRPC request", "request", msgIn, "response", outjson)
		return out, nil
	}, nil
}

func parseMessage(ref protoreflect.Message) map[string]interface{} {
	result := make(map[string]interface{})
	if ref == nil {
		return result
	}

	ref.Range(func(fd protoreflect.FieldDescriptor, v protoreflect.Value) bool {
		key := string(fd.Name())
		result[key] = getFieldValue(v, fd)
		return true
	})

	return result
}

func getFieldValue(v protoreflect.Value, fd protoreflect.FieldDescriptor) interface{} {
	switch {
	case fd.IsList():
		return processList(v.List(), fd)
	case fd.IsMap():
		return processMap(v.Map(), fd)
	case fd.Kind() == protoreflect.MessageKind:
		return parseMessage(v.Message())
	case fd.Kind() == protoreflect.EnumKind:
		return fd.Enum().Values().ByNumber(v.Enum()).Name()
	default:
		return v.Interface()
	}
}

func processList(list protoreflect.List, fd protoreflect.FieldDescriptor) []interface{} {
	res := make([]interface{}, list.Len())
	for i := 0; i < list.Len(); i++ {
		switch fd.Kind() { //nolint:exhaustive
		case protoreflect.MessageKind:
			res[i] = parseMessage(list.Get(i).Message())
		case protoreflect.EnumKind:
			res[i] = fd.Enum().Values().ByNumber(list.Get(i).Enum()).Name()
		case protoreflect.StringKind:
			res[i] = list.Get(i).String()
		case protoreflect.Int32Kind, protoreflect.Int64Kind, protoreflect.Sint32Kind, protoreflect.Sint64Kind:
			res[i] = list.Get(i).Int()
		case protoreflect.Uint32Kind, protoreflect.Uint64Kind:
			res[i] = list.Get(i).Uint()
		case protoreflect.BoolKind:
			res[i] = list.Get(i).Bool()
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			res[i] = list.Get(i).Float()
		case protoreflect.BytesKind:
			res[i] = list.Get(i).Bytes()
		default:
			res[i] = list.Get(i).Interface()
		}
	}
	return res
}

func processMap(m protoreflect.Map, fd protoreflect.FieldDescriptor) map[string]interface{} {
	res := make(map[string]interface{})

	m.Range(func(k protoreflect.MapKey, v protoreflect.Value) bool {
		res[k.String()] = getFieldValue(v, fd.MapValue())
		return true
	})

	return res
}
