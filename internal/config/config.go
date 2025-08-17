package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"time"
)

type Config struct {
	Bitcoin   BitcoinConfig   `yaml:"bitcoin"`
	Database  DatabaseConfig  `yaml:"database"`
	Ingestion IngestionConfig `yaml:"ingestion"`
}

type BitcoinConfig struct {
	RPC RPCConfig `yaml:"rpc"`
}

type RPCConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	TLS      bool   `yaml:"tls"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type IngestionConfig struct {
	StartDate    string        `yaml:"start_date"`
	BatchSize    int           `yaml:"batch_size"`
	PollInterval time.Duration `yaml:"poll_interval"`
}

func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

func (r *RPCConfig) URL() string {
	scheme := "http"
	if r.TLS {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s:%s@%s:%d", scheme, r.User, r.Password, r.Host, r.Port)
}