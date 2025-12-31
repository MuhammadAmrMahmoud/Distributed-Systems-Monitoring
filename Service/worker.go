package service

import (
	"Distributed-Health-Monitoring/Repository"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/streadway/amqp"
)

func (e *Engine) AMQPURL() string {
	r := e.Cnfg.RabbitMQ
	vhost := r.VHost
	if vhost == "" {
		vhost = "/"
	}
	return fmt.Sprintf(
		"amqp://%s:%s@%s:%d%s",
		r.Username,
		r.Password,
		r.Host,
		r.Port,
		vhost,
	)
}

func (e *Engine) StartWorker(amqpURL, queueName string) error {
	conn, err := amqp.Dial(amqpURL)
	if err != nil {
		return err
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	msgs, err := ch.Consume(
		queueName,
		"",
		false, 
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return err
	}

	for msg := range msgs {
		var job HealthCheckJob
		if err := json.Unmarshal(msg.Body, &job); err != nil {
			log.Printf("[WORKER] invalid_job err=%v", err)
			msg.Nack(false, false)
			continue
		}

		// Load service from DB
		service, err := e.Repo.GetServiceByName(context.Background(), job.ServiceName)
		if err != nil {
			log.Printf("[WORKER] service_not_found service=%s", job.ServiceName)
			msg.Nack(false, false)
			continue
		}

		req, err := http.NewRequest(
			job.Method,
			job.URL, 
			nil,
		)
		if err != nil {
			log.Printf("[WORKER] invalid_request service=%s err=%v", service.Name, err)
			msg.Nack(false, false)
			continue
		}

		client := &http.Client{
			Timeout: job.Timeout,
		}

		start := time.Now()
		resp, err := client.Do(req)
		latencyMs := time.Since(start).Milliseconds()

		status := "DOWN"
		statusCode := 0
		errorMsg := ""
		success := false

		if err != nil {
			errorMsg = err.Error()
		} else {
			defer resp.Body.Close()
			statusCode = resp.StatusCode
			if resp.StatusCode < 400 {
				status = "UP"
				success = true
			}
		}

		// Save append-only log
		if err := e.Repo.SaveServiceCheckLog(
			*service,
			status,
			statusCode,
			latencyMs,
			errorMsg,
		); err != nil {
			log.Printf("[WORKER] log_save_failed service=%s err=%v", service.Name, err)
		}

		// Update service state
		stateChange, err := e.Repo.UpdateServiceState(context.Background(), service, success)
		if err != nil {
			log.Printf("[WORKER] state_update_failed service=%s err=%v", service.Name, err)
		}

		// 🔹 Broadcast only on transition
		if stateChange != nil {
			LogStateTransition(service.Name, stateChange) // Log the transition in the db
			BroadcastStateChange(*service, stateChange) // Broadcast the transition with the WebSocket endpoint
		}

		log.Printf(
			"[WORKER] check_completed service=%s status=%s latency_ms=%d error=%s",
			service.Name,
			status,
			latencyMs,
			errorMsg,
		)

		// Acknowledge only after successful processing
		msg.Ack(false)
	}

	return nil
}

func LogStateTransition(serviceName string, change *Repository.StateChange) {
	log.Printf(
		"[STATE_TRANSITION] service=%s from=%s to=%s at=%s",
		serviceName,
		change.From,
		change.To,
		time.Now().Format(time.RFC3339),
	)
}
