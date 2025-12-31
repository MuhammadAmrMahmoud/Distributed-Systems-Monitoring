package config

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Config holds the structure of config.json
type Config struct {
	PostgreSQL PostgreSQL `json:"postgresql"`
	RabbitMQ   RabbitMQ   `json:"rabbitmq"`
	Server     Server     `json:"server"`
	Auth       AuthConfig `json:"auth"`
}

type PostgreSQL struct {
	Host         string `json:"host"`
	Port         int    `json:"port"`
	User         string `json:"user"`
	Password     string `json:"password"`
	Database     string `json:"database"`
	SSLMode      string `json:"sslmode"`
	MaxOpenConns int    `json:"max_open_conns"`
	MaxIdleConns int    `json:"max_idle_conns"`
}

type RabbitMQ struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	VHost      string `json:"vhost"`
	QueueName  string `json:"queue_name"`
	Exchange   string `json:"exchange"`
	RoutingKey string `json:"routing_key"`
}

type Server struct {
	Address string `json:"address"`
}

type AuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}


// LoadConfig reads the config file and unmarshals it
func LoadConfig(filePath string) (*Config, error) {

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := json.Unmarshal(bytes, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	return &config, nil
}

// ConnectPostgres establishes a PostgreSQL connection using GORM
func ConnectPostgres(cfg *Config) (*gorm.DB, error) {
	pgCfg := cfg.PostgreSQL
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		pgCfg.Host, pgCfg.Port, pgCfg.User, pgCfg.Password, pgCfg.Database, pgCfg.SSLMode,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB from GORM: %w", err)
	}

	sqlDB.SetMaxOpenConns(pgCfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(pgCfg.MaxIdleConns)

	return db, nil
}

func GetServerAddress(cfg *Config) string {
	return cfg.Server.Address
}
