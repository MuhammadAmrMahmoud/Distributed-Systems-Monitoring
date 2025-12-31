package service

import (
	"Distributed-Health-Monitoring/config"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
)

// HealthCheckJob represents a job to check a service
type HealthCheckJob struct {
	ServiceName string        `json:"service_name"`
	URL         string        `json:"url"`
	Timeout     time.Duration `json:"timeout"`
	Method      string        `json:"method"`
}

// Scheduler handles scheduling health checks
type Scheduler struct {
	amqpConn    *amqp.Connection
	amqpChannel *amqp.Channel
	queueName   string
}

// NewScheduler connects to RabbitMQ and returns a Scheduler
func (e *Engine) NewScheduler(cfg *config.Config) (*Scheduler, error) {
	// Get RabbitMQ configuration
	rbtCnfg := cfg.RabbitMQ

	url := fmt.Sprintf("amqp://%s:%s@%s:%d%s",
		rbtCnfg.Username, rbtCnfg.Password, rbtCnfg.Host, rbtCnfg.Port, rbtCnfg.VHost)
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	// declare the queue
	_, err = ch.QueueDeclare(
		rbtCnfg.QueueName,
		true,  // durable
		false, // autoDelete
		false, // exclusive
		false, // noWait
		nil,   // args
	)
	if err != nil {
		return nil, fmt.Errorf("failed to declare queue: %w", err)
	}

	return &Scheduler{
		amqpConn:    conn,
		amqpChannel: ch,
		queueName:   rbtCnfg.QueueName,
	}, nil
}

// Schedule adds a health check job to the queue
func (s *Scheduler) Schedule(job HealthCheckJob) error {
	body, err := json.Marshal(job)
	if err != nil {
		LogJobScheduleError(job, err)
		return fmt.Errorf("failed to marshal job: %w", err)
	}

	err = s.amqpChannel.Publish(
		"",
		s.queueName,
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
			Timestamp:   time.Now(),
		},
	)
	if err != nil {
		LogJobScheduleError(job, err)
		return fmt.Errorf("failed to publish job: %w", err)
	}

	LogJobScheduled(job)
	return nil
}

// Close cleans up connections
func (s *Scheduler) Close() {
	s.amqpChannel.Close()
	s.amqpConn.Close()
}

func LogJobScheduled(job HealthCheckJob) {
	log.Printf(
		"[SCHEDULER] job_scheduled service=%s method=%s url=%s timeout=%s at=%s",
		job.ServiceName,
		job.Method,
		job.URL,
		job.Timeout,
		time.Now().Format(time.RFC3339),
	)
}

func LogJobScheduleError(job HealthCheckJob, err error) {
	log.Printf(
		"[SCHEDULER] job_schedule_failed service=%s error=%v",
		job.ServiceName,
		err,
	)
}