package main

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	status "google.golang.org/grpc/status"
)

func (s *MyMicroserviceData) ACLUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("unary: failed to get metadata from context")
	}

	fullMethod := info.FullMethod
	consumer := meta.Get("consumer")[0]

	peer, _ := peer.FromContext(ctx)
	addr := peer.Addr.String()

	if err := s.aclValidator(consumer, fullMethod); err != nil {
		return nil, fmt.Errorf("unary: %w", err)
	}

	s.chanMu.Lock()
	for _, value := range s.consumersChans {
		value <- &Event{
			Consumer: consumer,
			Method:   info.FullMethod,
			Host:     addr,
		}
	}
	s.chanMu.Unlock()
	return handler(ctx, req)
}

func (s *MyMicroserviceData) ACLStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	meta, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "stream: failed to get metadata from context")
	}

	fullMethod := info.FullMethod
	consumer := meta.Get("consumer")[0]

	if err := s.aclValidator(consumer, fullMethod); err != nil {
		return fmt.Errorf("stream: %w", err)
	}

	peer, _ := peer.FromContext(ss.Context())
	addr := peer.Addr.String()

	s.chanMu.Lock()
	for _, value := range s.consumersChans {
		select {
		case value <- &Event{
			Consumer: consumer,
			Method:   info.FullMethod,
			Host:     addr,
		}:
		default:
		}
	}
	s.chanMu.Unlock()

	return handler(srv, ss)
}

func (s *MyMicroserviceData) aclValidator(consumer string, fullMethod string) error {
	if consumer == "" {
		return status.Errorf(codes.Unauthenticated, "consumer not found")
	}

	if _, ok := (s.acl)[consumer]; !ok {
		return status.Errorf(codes.Unauthenticated, "consumer not found in ACL")
	}

	allowed := false
	for _, method := range s.acl[consumer] {
		if strings.HasSuffix(method, "/*") {
			allowed = strings.HasPrefix(fullMethod, strings.TrimSuffix(method, "*"))
			break
		}
		if method == fullMethod {
			allowed = true
			break
		}
	}
	if !allowed {
		return status.Errorf(codes.Unauthenticated, "method not allowed")
	}
	return nil
}
