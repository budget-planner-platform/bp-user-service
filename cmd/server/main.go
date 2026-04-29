package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"buf.build/gen/go/budget-planner-platform/bp-user/grpc/go/user_service/v1/user_servicev1grpc"
	"github.com/joho/godotenv"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/yaroslav/bp-user-service/internal/db"
	"github.com/yaroslav/bp-user-service/internal/handler"
	"github.com/yaroslav/bp-user-service/internal/repository"
)

func main() {
	_ = godotenv.Load()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	repo := repository.NewUserRepository(pool)
	userHandler := handler.NewUserHandler(repo)

	srv := grpc.NewServer()
	user_servicev1grpc.RegisterUserServiceServer(srv, userHandler)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", port, err)
	}

	go func() {
		<-ctx.Done()
		log.Println("Shutting down gRPC server...")
		srv.GracefulStop()
	}()

	log.Printf("gRPC server listening on :%s", port)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
