package grpc

import (
	"Distributed-Health-Monitoring/models"
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// CheckGRPCWithLatency returns health status and connection latency
func Check_gRPC(address string, timeout time.Duration) models.GRPCHealthResult {
	startTime := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	
	latency := time.Since(startTime)
	
	if err != nil {
		return models.GRPCHealthResult{
			IsHealthy:  false,
			Latency:    latency,
			StatusCode: connectivity.TransientFailure,
			Error:      fmt.Errorf("connection failed: %w", err),
		}
	}
	defer conn.Close()

	state := conn.GetState()
	isHealthy := state == connectivity.Ready

	return models.GRPCHealthResult{
		IsHealthy:  isHealthy,
		Latency:    latency,
		StatusCode: state,
		Error:      nil,
	}
}