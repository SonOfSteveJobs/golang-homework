package main

import (
	context "context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	sync "sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type AdminService struct {
	UnimplementedAdminServer
	data *MyMicroserviceData
}

type BizService struct {
	UnimplementedBizServer
}

type MyMicroserviceData struct {
	acl            map[string][]string
	consumersChans map[int]chan *Event
	nextID         int
	chanMu         sync.Mutex
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	data := &MyMicroserviceData{consumersChans: make(map[int]chan *Event), chanMu: sync.Mutex{}}
	if err := json.Unmarshal([]byte(ACLData), &data.acl); err != nil {
		return fmt.Errorf("failed to parse ACL data: %v", err)
	}

	server := grpc.NewServer(
		grpc.UnaryInterceptor(data.ACLUnaryInterceptor),
		grpc.StreamInterceptor(data.ACLStreamInterceptor),
	)

	adminService := &AdminService{data: data}
	bizService := &BizService{}
	RegisterAdminServer(server, adminService)
	RegisterBizServer(server, bizService)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// для постмана
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

func (s *AdminService) Logging(_ *Nothing, adm Admin_LoggingServer) error {
	mu := &s.data.chanMu

	mu.Lock()
	id := s.data.nextID
	ch := make(chan *Event, 1)

	s.data.nextID++
	s.data.consumersChans[id] = ch
	mu.Unlock()

	for {
		select {
		case <-adm.Context().Done():
			mu.Lock()
			delete(s.data.consumersChans, id)
			mu.Unlock()
			return nil
		case event := <-ch:
			adm.Send(event)
		}
	}
}
func (s *AdminService) Statistics(*StatInterval, Admin_StatisticsServer) error {
	return nil
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
