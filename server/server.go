package server

import (
	"errors"
	"fmt"
	"log/slog"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/default23/protofake/config"
	"github.com/default23/protofake/mapper"
)

// Server is the gRPC mocking server.
type Server struct {
	config     config.GRPC
	grpcServer *grpc.Server
	listener   net.Listener
	services   map[string]*ServiceDesc

	mappings       map[string][]mapper.Mapping
	messageFactory map[string]MessageFactory
}

// New - creates a new gRPC mocking server.
func New(conf config.GRPC) (*Server, error) {
	srv := grpc.NewServer()

	listener, err := net.Listen("tcp", net.JoinHostPort(conf.Host, conf.Port))
	if err != nil {
		return nil, fmt.Errorf("construct gRPC server: listen %s:%s: %w", conf.Host, conf.Port, err)
	}

	return &Server{
		config:         conf,
		grpcServer:     srv,
		listener:       listener,
		services:       make(map[string]*ServiceDesc),
		messageFactory: make(map[string]MessageFactory),
	}, nil
}

func (s *Server) SetMappings(mappings []mapper.Mapping) {
	endpointMappings := make(map[string][]mapper.Mapping)
	for _, m := range mappings {
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
}

// Close - gracefully shuts down the gRPC server.
func (s *Server) Close() error {
	s.grpcServer.GracefulStop()
	if err := s.listener.Close(); err != nil && !errors.Is(err, net.ErrClosed) {
		return fmt.Errorf("close listener: %w", err)
	}

	return nil
}

// Run - starts the gRPC server.
func (s *Server) Run() {
	slog.Info("starting gRPC server at " + s.config.Host + ":" + s.config.Port)
	go func() {
		if s.config.ServerReflection {
			reflection.Register(s.grpcServer)
		}
		if err := s.grpcServer.Serve(s.listener); err != nil {
			slog.Error("failed to start gRPC server", "error", err)
		}
	}()
}
