package main

import (
	service "Distributed-Health-Monitoring/Service"
	"context"
	"log"
)

func main() {

	engine, err := service.NewEngine()
	if err != nil {
		log.Fatalf("Failed to create engine: %v", err)
	}

	// Setup all routes
	engine.SetupRoutes()

	// WebSocket hub
	hub := engine.NewHub()
	service.GlobalHub = hub
	
	// START WEBSOCKET
	go hub.Run()

	// START WORKER
	go func() {
		if err := engine.StartWorker(engine.AMQPURL(), engine.Cnfg.RabbitMQ.QueueName); err != nil {
			log.Fatalf("worker failed: %v", err)
		}
	}()

	// START SCHEDULER
	go func() {
		if err := engine.Scheduler(context.Background()); err != nil {
			log.Fatalf("scheduler failed: %v", err)
		}
	}()

	// START GIN SERVER
	if err := engine.Run(); err != nil {
		log.Fatalf("Failed to run engine: %v", err)
	}

}
