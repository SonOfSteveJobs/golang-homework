package main

import (
	context "context"
	"encoding/json"
	"fmt"
	"net"
	sync "sync"
	"time"

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
	chanMu         sync.Mutex
	nextID         int
	stats          map[int]*Stat
	nextStatID     int
	statsMu        sync.Mutex
}

// тут вы пишете код
// обращаю ваше внимание - в этом задании запрещены глобальные переменные
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	data := &MyMicroserviceData{
		consumersChans: make(map[int]chan *Event),
		chanMu:         sync.Mutex{},
		stats:          make(map[int]*Stat),
		statsMu:        sync.Mutex{},
	}

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
		return fmt.Errorf("failed to listen: %v", err)
	}

	// для постмана
	reflection.Register(server)

	go func() {
		if err := server.Serve(lis); err != nil {
			fmt.Printf("server serve error: %v", err)
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
	ch := make(chan *Event, 10)

	s.data.nextID++
	s.data.consumersChans[id] = ch
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(s.data.consumersChans, id)
		mu.Unlock()
	}()

	for {
		select {
		case <-adm.Context().Done():
			return nil
		case event := <-ch:
			if err := adm.Send(event); err != nil {
				return err
			}
		}
	}
}
func (s *AdminService) Statistics(interval *StatInterval, adm Admin_StatisticsServer) error {
	mu := &s.data.statsMu

	mu.Lock()
	id := s.data.nextStatID
	s.data.nextStatID++
	s.data.stats[id] = &Stat{
		ByMethod:   make(map[string]uint64),
		ByConsumer: make(map[string]uint64),
	}
	mu.Unlock()

	defer func() {
		mu.Lock()
		delete(s.data.stats, id)
		mu.Unlock()
	}()

	ticker := time.NewTicker(time.Duration(interval.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-adm.Context().Done():
			return nil
		case <-ticker.C:
			mu.Lock()
			stat := s.data.stats[id]
			s.data.stats[id] = &Stat{
				ByMethod:   make(map[string]uint64),
				ByConsumer: make(map[string]uint64),
			}
			mu.Unlock()
			if err := adm.Send(stat); err != nil {
				return err
			}
		}
	}
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
