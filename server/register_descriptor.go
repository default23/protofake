package server

import (
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
)

// MockServer is a placeholder for the gRPC service implementation.
type MockServer interface{}

type ServiceDesc struct {
	Service           *grpc.ServiceDesc
	ServiceDescriptor *descriptorpb.ServiceDescriptorProto
	FileDescriptor    *descriptorpb.FileDescriptorProto
}

// Register - registers provided gRPC services.
func (s *Server) Register(descriptor *descriptorpb.FileDescriptorSet) error {
	protos := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, fd := range descriptor.File {
		protos[fd.GetName()] = fd
	}

	var registerProtoFile func(name string) error
	alreadyProcessed := make(map[string]struct{})
	registerProtoFile = func(name string) error {
		if _, ok := alreadyProcessed[name]; ok {
			return nil
		}
		alreadyProcessed[name] = struct{}{}

		proto := protos[name]
		for _, dep := range proto.Dependency {
			if err := registerProtoFile(dep); err != nil {
				return err
			}
		}

		f, _ := protoregistry.GlobalFiles.FindFileByPath(proto.GetName())
		if f != nil {
			return nil // already registered
		}

		fd, err := protodesc.NewFile(proto, protoregistry.GlobalFiles)
		if err != nil {
			return fmt.Errorf("create %s descriptor file: %w", name, err)
		}
		if err = protoregistry.GlobalFiles.RegisterFile(fd); err != nil {
			return fmt.Errorf("register %s descriptor file: %w", name, err)
		}

		return nil
	}
	for name := range protos {
		if err := registerProtoFile(name); err != nil {
			return fmt.Errorf("register %s descriptor file: %w", name, err)
		}
	}

	for _, proto := range protos {
		if len(proto.GetService()) == 0 {
			continue
		}

		for _, service := range proto.GetService() {
			sd, err := s.NewServiceDesc(proto, service)
			if err != nil {
				return fmt.Errorf("construct service '%s' desc: %w", service.GetName(), err)
			}

			if _, ok := s.services[sd.ServiceName]; ok {
				if s.config.IgnoreDuplicateService {
					slog.Warn("service already registered, ignore_duplicate_service option is ON, so it will be ignored and not handled", "name", sd.ServiceName)
					continue
				}

				return fmt.Errorf("service %s already registered", sd.ServiceName)
			}

			s.services[sd.ServiceName] = &ServiceDesc{
				Service:           sd,
				ServiceDescriptor: service,
				FileDescriptor:    proto,
			}

			var methodNames []string
			for _, method := range service.GetMethod() {
				methodNames = append(methodNames, fmt.Sprintf("%s/%s", sd.ServiceName, method.GetName()))
			}
			slog.Debug("registered service", "full_name", sd.ServiceName, "package", proto.GetPackage(), "file", proto.GetName(), "methods", methodNames)
		}
	}

	for _, service := range s.services {
		s.grpcServer.RegisterService(service.Service, MockServer(s))
	}

	return nil
}

// NewServiceDesc constructs the gRPC service desc, is needed to handle incoming requests.
func (s *Server) NewServiceDesc(protoDescriptor *descriptorpb.FileDescriptorProto, descr *descriptorpb.ServiceDescriptorProto) (*grpc.ServiceDesc, error) {
	out := &grpc.ServiceDesc{
		ServiceName: fmt.Sprintf("%s.%s", protoDescriptor.GetPackage(), descr.GetName()),
		HandlerType: (*MockServer)(nil),
		Metadata:    protoDescriptor.Name,
	}

	for _, method := range descr.GetMethod() {
		handler, err := s.NewMockHandler(protoDescriptor, descr, method)
		if err != nil {
			return nil, fmt.Errorf("construct method '%s/%s' handler: %w", out.ServiceName, method.GetName(), err)
		}

		desc := grpc.MethodDesc{
			MethodName: method.GetName(),
			Handler:    handler,
		}
		out.Methods = append(out.Methods, desc)
	}

	return out, nil
}
