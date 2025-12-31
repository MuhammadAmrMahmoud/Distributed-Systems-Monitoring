package models

import (
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc/connectivity"
)

// ExternalService represents a service to be monitored
type ExternalService struct {
	ID                  uint       `json:"id" gorm:"primaryKey;autoIncrement"`
	Name                string     `json:"name" gorm:"type:varchar(255);not null;uniqueIndex"`
	URL                 string     `json:"url" gorm:"type:varchar(500);not null"`
	HTTPMethod          string     `json:"http_method" gorm:"type:varchar(10);not null;default:'GET'"`
	Protocol            string     `json:"protocol" gorm:"type:varchar(10);not null;default:'HTTP'"`
	Interval            int64      `json:"interval" gorm:"type:bigint;not null;default:60"` // check interval in seconds
	TimeoutSeconds      int64      `json:"timeout_seconds" gorm:"type:bigint;not null;default:10"`
	FailureThreshold    int64      `json:"failure_threshold" gorm:"type:bigint;not null;default:3"`    // consecutive failures before marking as down
	Status              string     `json:"status" gorm:"type:varchar(20);not null;default:'up';index"` // "up" or "down"
	ConsecutiveFailures int64      `json:"consecutive_failures" gorm:"type:bigint;not null;default:0"`
	LastCheckedAt       *time.Time `json:"last_checked_at" gorm:"type:timestamp"`
	CreatedAt           time.Time  `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt           time.Time  `json:"updated_at" gorm:"autoUpdateTime"`
}

// ServiceCheckLog records the result of each health check
type ServiceCheckLog struct {
	ID                uint            `json:"id" gorm:"primaryKey;autoIncrement"`
	ExternalServiceID uint            `json:"external_service_id" gorm:"not null;index:idx_service_time"`
	Status            string          `json:"status" gorm:"type:varchar(20);not null;default:'up'"` // up, down, timeout, error
	StatusCode        int             `json:"status_code" gorm:"type:int"`                          // HTTP status code
	ResponseTimeMs    int64           `json:"response_time_ms" gorm:"type:bigint"`                  // response time in milliseconds
	ErrorMessage      string          `json:"error_message,omitempty" gorm:"type:text"`
	CheckedAt         time.Time       `json:"checked_at" gorm:"type:timestamp;not null;index:idx_service_time"`
	ExternalService   ExternalService `json:"-" gorm:"foreignKey:ExternalServiceID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

type StateChange struct {
	From string
	To   string
}

type Client struct {
	Conn *websocket.Conn
	Send chan []byte
}

type ServiceStateChangeEvent struct {
	Type      string    `json:"type"` // service_state_change
	ServiceID uint      `json:"service_id"`
	Name      string    `json:"name"`
	From      string    `json:"from"`
	To        string    `json:"to"`
	Timestamp time.Time `json:"timestamp"`
}

type GRPCHealthResult struct {
	IsHealthy  bool
	Latency    time.Duration
	StatusCode connectivity.State // IDLE, CONNECTING, READY, TRANSIENT_FAILURE, SHUTDOWN
	Error      error
}

// TableName specifies the table name for ExternalService
func (ExternalService) TableName() string {
	return "external_services"
}

// TableName specifies the table name for ServiceCheckLog
func (ServiceCheckLog) TableName() string {
	return "service_check_logs"
}

// ShouldMarkDown determines if the service should be marked as down
func (s *ExternalService) ShouldMarkDown() bool {
	return s.ConsecutiveFailures >= s.FailureThreshold
}

// RecordSuccess resets the consecutive failures counter
func (s *ExternalService) RecordSuccess() {
	s.Status = "UP"
	s.ConsecutiveFailures = 0
	now := time.Now()
	s.LastCheckedAt = &now
}

// RecordFailure increments the consecutive failures counter
func (s *ExternalService) RecordFailure() {
	s.ConsecutiveFailures++
	now := time.Now()
	s.LastCheckedAt = &now

	if s.ShouldMarkDown() {
		s.Status = "DOWN"
	}
}
