package main

import (
	context "context"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type AdminService struct {
	UnimplementedAdminServer
}

type BizService struct {
	UnimplementedBizServer
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	server := grpc.NewServer()
	adminService := &AdminService{}
	bizService := &BizService{}

	RegisterAdminServer(server, adminService)
	RegisterBizServer(server, bizService)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
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
