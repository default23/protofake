package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
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

		ua := strings.Join(md.Get("user-agent"), ";")
		xreq := strings.Join(md.Get("x-request-id"), ";")
		if xreq == "" {
			xreq = uuid.NewString()
		}

		logger := slog.With(
			"method", fullMethodName,
			"input_type", methodDescr.GetInputType(),
			"output_type", methodDescr.GetOutputType(),
			"metadata", md,
			"user-agent", ua,
			"x-request-id", xreq,
		)

		in, out := msgFactory()
		if err = dec(in); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode input message: %v", err))
		}

		var jv []byte
		jv, err = protojson.MarshalOptions{UseProtoNames: true}.Marshal(in.Interface())
		slog.Debug("marshalled input message", "input", string(jv))
		if err != nil {
			slog.Error("marshalling input message", "error", err)
		}

		msgIn := make(map[string]any)
		if err = json.Unmarshal(jv, &msgIn); err != nil {
			return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to unmarshal input message: %v", err))
		}

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

		var outValue []byte
		outValue, err = buildResponse(shouldRespondWith, msgIn, md)
		if err != nil {
			return nil, err
		}

		unmarshalOpts := protojson.UnmarshalOptions{
			DiscardUnknown: s.config.DiscardUnknownFields,
		}
		if err = unmarshalOpts.Unmarshal(outValue, out.Interface()); err != nil {
			return nil, status.Errorf(codes.FailedPrecondition, "check registered mappings for method, failed to unmarshal output message into %s message: %v", methodDescr.GetOutputType(), err.Error())
		}

		logger.Debug("successfully mapped gRPC request", "request", msgIn, "response", string(outValue))
		return out, nil
	}, nil
}
