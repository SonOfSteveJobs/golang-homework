package main

import (
	"context"
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
		return nil, status.Errorf(codes.Unauthenticated, "unary: failed to get metadata from context")
	}

	fullMethod := info.FullMethod

	consumers := meta.Get("consumer")
	if len(consumers) == 0 {
		return nil, status.Errorf(codes.Unauthenticated, "consumer not found")
	}
	consumer := consumers[0]

	peer, _ := peer.FromContext(ctx)
	addr := peer.Addr.String()

	if err := s.aclValidator(consumer, fullMethod); err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "unary: %v", err)
	}

	s.statsMu.Lock()
	for _, value := range s.stats {
		value.ByConsumer[consumer]++
		value.ByMethod[fullMethod]++
	}
	s.statsMu.Unlock()

	s.chanMu.Lock()
	for _, ch := range s.consumersChans {
		ch <- &Event{
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

	consumers := meta.Get("consumer")
	if len(consumers) == 0 {
		return status.Errorf(codes.Unauthenticated, "consumer not found")
	}
	consumer := consumers[0]

	if err := s.aclValidator(consumer, fullMethod); err != nil {
		return status.Errorf(codes.Unauthenticated, "stream: %v", err)
	}

	s.statsMu.Lock()
	for _, value := range s.stats {
		value.ByConsumer[consumer]++
		value.ByMethod[fullMethod]++
	}
	s.statsMu.Unlock()

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
