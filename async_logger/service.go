package main

import (
	context "context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"strings"

	"google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	status "google.golang.org/grpc/status"
)

type AdminService struct {
	UnimplementedAdminServer
}

type BizService struct {
	UnimplementedBizServer
}

type MyMicroserviceData struct {
	acl map[string][]string
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	data := &MyMicroserviceData{}
	if err := json.Unmarshal([]byte(ACLData), &data.acl); err != nil {
		return fmt.Errorf("failed to parse ACL data: %v", err)
	}
	server := grpc.NewServer(grpc.UnaryInterceptor(data.ACLUnaryInterceptor), grpc.StreamInterceptor(data.ACLStreamInterceptor))
	adminService := &AdminService{}
	bizService := &BizService{}

	RegisterAdminServer(server, adminService)
	RegisterBizServer(server, bizService)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	reflection.Register(server)

	go func() {
		err := server.Serve(lis)
		if err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	go func() {
		<-ctx.Done()
		server.GracefulStop()
	}()

	return nil
}

func (s *MyMicroserviceData) ACLUnaryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Printf("ACL interceptor \n ctx: %v\n, req: %v \n, info: %v \n, handler: %v \n", ctx, req, info, handler)
	meta, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, fmt.Errorf("failed to get metadata from context")
	}
	fmt.Printf("consumer: %v", meta.Get("consumer"))

	consumer := meta.Get("consumer")
	if consumer == nil {
		return nil, status.Errorf(codes.Unauthenticated, "consumer not found")
	}

	if _, ok := (s.acl)[consumer[0]]; !ok {
		return nil, status.Errorf(codes.Unauthenticated, "consumer not found in ACL")
	}

	allowed := false
	for _, method := range s.acl[consumer[0]] {
		if strings.HasSuffix(method, "/*") {
			allowed = strings.HasPrefix(info.FullMethod, strings.TrimSuffix(method, "*"))
			break
		}
		if method == info.FullMethod {
			allowed = true
			break
		}
	}
	if !allowed {
		return nil, status.Errorf(codes.Unauthenticated, "method not allowed")
	}

	return handler(ctx, req)
}

func (*BizService) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
func (*BizService) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}
func (*BizService) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *MyMicroserviceData) ACLStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo,
	handler grpc.StreamHandler) error {
	meta, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.Unauthenticated, "no metadata")
	}

	consumer := meta.Get("consumer")
	if consumer == nil {
		return status.Errorf(codes.Unauthenticated, "consumer not found")
	}

	if _, ok := (s.acl)[consumer[0]]; !ok {
		return status.Errorf(codes.Unauthenticated, "consumer not found in ACL")
	}

	allowed := false
	for _, method := range s.acl[consumer[0]] {
		if strings.HasSuffix(method, "/*") {
			allowed = strings.HasPrefix(info.FullMethod, strings.TrimSuffix(method, "*"))
			break
		}
		if method == info.FullMethod {
			allowed = true
			break
		}
	}
	if !allowed {
		return status.Errorf(codes.Unauthenticated, "method not allowed")
	}

	return handler(srv, ss)
}

func (s *AdminService) Logging(*Nothing, Admin_LoggingServer) error {
	return nil
}
func (s *AdminService) Statistics(*StatInterval, Admin_StatisticsServer) error {
	return nil
}
