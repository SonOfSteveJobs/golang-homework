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
	consumer, err := s.getConsumerFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := s.aclValidator(consumer, info.FullMethod); err != nil {
		return nil, err
	}

	peer, _ := peer.FromContext(ctx)
	addr := peer.Addr.String()

	s.updateStats(consumer, info.FullMethod)
	s.broadcastEvent(consumer, info.FullMethod, addr)

	return handler(ctx, req)
}

func (s *MyMicroserviceData) ACLStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	consumer, err := s.getConsumerFromCtx(ss.Context())
	if err != nil {
		return err
	}

	if err := s.aclValidator(consumer, info.FullMethod); err != nil {
		return err
	}

	peer, _ := peer.FromContext(ss.Context())
	addr := peer.Addr.String()

	s.updateStats(consumer, info.FullMethod)
	s.broadcastEvent(consumer, info.FullMethod, addr)

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

func (s *MyMicroserviceData) getConsumerFromCtx(ctx context.Context) (string, error) {
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", status.Errorf(codes.Unauthenticated, "no metadata in context")
	}
	consumers := meta.Get("consumer")
	if len(consumers) == 0 {
		return "", status.Errorf(codes.Unauthenticated, "consumer not found")
	}
	return consumers[0], nil
}

func (s *MyMicroserviceData) broadcastEvent(consumer, fullMethod, addr string) {
	s.chanMu.Lock()
	for _, ch := range s.consumersChans {
		select {
		case ch <- &Event{
			Consumer: consumer,
			Method:   fullMethod,
			Host:     addr,
		}:
		default:
		}
	}
	s.chanMu.Unlock()
}

func (s *MyMicroserviceData) updateStats(consumer, fullMethod string) {
	s.statsMu.Lock()
	for _, value := range s.stats {
		value.ByConsumer[consumer]++
		value.ByMethod[fullMethod]++
	}
	s.statsMu.Unlock()
}
