package Repository

import (
	"Distributed-Health-Monitoring/cache"
	"Distributed-Health-Monitoring/models"
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

type DbRepository struct {
	db *gorm.DB
	IRepository
}

type IRepository interface {
	RegisterService(ctx context.Context, service *models.ExternalService) error
	GetAllServices(ctx context.Context) (map[uint]*models.ExternalService, error)
	SaveServiceCheckLog(service models.ExternalService, status string, statusCode int, responseTimeMs int64, errMsg string) error
	UpdateServiceState(ctx context.Context, service *models.ExternalService, success bool) (*StateChange, error)
	GetServiceByName(ctx context.Context, name string) (*models.ExternalService, error)
	GetServiceCheckLogs(ctx context.Context, serviceID uint, limit int, offset int) ([]*models.ServiceCheckLog, error)
}

func NewRepository(db *gorm.DB) IRepository {
	return &DbRepository{
		db: db,
	}
}

func (r *DbRepository) RegisterService(ctx context.Context, service *models.ExternalService) error {

	// some validations
	if service == nil {
		return errors.New("service is nil")
	}
	if service.Name == "" {
		return errors.New("service name is empty")
	}
	if service.URL == "" {
		return errors.New("service url is empty")
	}
	if service.HTTPMethod == "" {
		return errors.New("service http method is empty")
	}
	if service.TimeoutSeconds == 0 || service.TimeoutSeconds < 0 {
		return errors.New("service timeout is invalid")
	}
	if service.HTTPMethod != "GET" && service.HTTPMethod != "POST" && service.HTTPMethod != "PUT" && service.HTTPMethod != "DELETE" && service.HTTPMethod != "PATCH" {
		return errors.New("service http method is invalid")
	}
	if service.FailureThreshold == 0 || service.FailureThreshold < 0 {
		return errors.New("service failure threshold is invalid")
	}
	if service.Interval == 0 || service.Interval < 0 {
		return errors.New("service interval is invalid")
	}

	return r.db.WithContext(ctx).Save(service).Error
}

func (r *DbRepository) GetAllServices(ctx context.Context) (map[uint]*models.ExternalService, error) {
	var services []*models.ExternalService

	if err := r.db.WithContext(ctx).Model(&models.ExternalService{}).Find(&services).Error; err != nil {
		return nil, err
	}

	if len(services) == 0 {
		return nil, errors.New("no services found")
	}

	for _, service := range services {
		cache.MapExternalServices[service.ID] = service
	}

	return cache.MapExternalServices, nil
}

func (r *DbRepository) GetServiceByName(ctx context.Context, name string) (*models.ExternalService, error) {
	var service models.ExternalService

	if err := r.db.WithContext(ctx).Model(&models.ExternalService{}).Where(&models.ExternalService{Name: name}).First(&service).Error; err != nil {
		return nil, err
	}

	return &service, nil
}

func (r *DbRepository) SaveServiceCheckLog(service models.ExternalService, status string, statusCode int, responseTimeMs int64, errMsg string) error {

	logEntry := models.ServiceCheckLog{
		ExternalServiceID: service.ID,
		Status:            status,
		StatusCode:        statusCode,
		ResponseTimeMs:    responseTimeMs,
		ErrorMessage:      errMsg,
		CheckedAt:         time.Now(),
	}

	return r.db.Create(&logEntry).Error
}

type StateChange struct {
	From string
	To   string
}

func (r *DbRepository) UpdateServiceState(ctx context.Context, service *models.ExternalService, success bool) (*StateChange, error) {

	previousStatus := service.Status

	if success {
		service.RecordSuccess()
	} else {
		service.RecordFailure()
	}

	if err := r.db.WithContext(ctx).Save(service).Error; err != nil {
		return nil, err
	}

	if previousStatus != service.Status {
		return &StateChange{
			From: previousStatus,
			To:   service.Status,
		}, nil
	}

	return nil, nil
}
func (r *DbRepository) GetServiceCheckLogs(ctx context.Context, serviceID uint, limit int, offset int) ([]*models.ServiceCheckLog, error) {
	var logs []*models.ServiceCheckLog

	if limit == 0 {
		limit = 100 // default limit
	}

	if err := r.db.WithContext(ctx).
		Where("external_service_id = ?", serviceID).
		Order("checked_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error; err != nil {
		return nil, err
	}

	return logs, nil
}