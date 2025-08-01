package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	pb "gRPC-rating/gen/github.com/ArtemKVD/gRPC-rating/gen"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"google.golang.org/grpc"
)

type server struct {
	pb.UnimplementedProductViewServiceServer
	producer *kafka.Producer
}

func (s *server) SendProductView(ctx context.Context, req *pb.ProductViewRequest) (*pb.ProductViewResponse, error) {

	topic := "product-views"
	deliveryChan := make(chan kafka.Event)

	err := s.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
		Value:          []byte(req.ProductId),
	}, deliveryChan)

	if err != nil {
		close(deliveryChan)
		log.Printf("error produce meassage")
		return nil, err
	}

	log.Printf("message delivered: %v", req.ProductId)
	return &pb.ProductViewResponse{Success: true}, nil
}

func main() {
	config := &kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092",
		"client.id":         "gRPC-backend",
		"acks":              "1",
	}

	producer, err := kafka.NewProducer(config)
	if err != nil {
		log.Fatalf("Failed to create Kafka producer: %v", err)
	}
	defer producer.Close()

	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterProductViewServiceServer(s, &server{producer: producer})

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server running on :50051")
		err := s.Serve(lis)
		if err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	<-stop
	s.GracefulStop()
	log.Println("Server stopped")
}
