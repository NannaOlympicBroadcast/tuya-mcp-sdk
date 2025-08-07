package config

import (
	"log"
	"os"

	"github.com/caarlos0/env"
	"gopkg.in/yaml.v3"
)

type Config struct {
	AccessId                string `json:"access_id" yaml:"access_id" env:"ACCESS_ID"`
	AccessSecret            string `json:"access_secret" yaml:"access_secret" env:"ACCESS_SECRET"`
	Endpoint                string `json:"endpoint" yaml:"endpoint" env:"ENDPOINT"`
	CustomMcpServerEndpoint string `json:"custom_mcp_server_endpoint" yaml:"custom_mcp_server_endpoint" env:"CUSTOM_MCP_SERVER_ENDPOINT"`
}

func InitializeConfig() *Config {
	cfg := &Config{}

	// 1. load config from yaml
	// 1.1. check if config file exists

	yamlPath := os.Getenv("CONFIG_PATH")
	if yamlPath == "" {
		yamlPath = "config.yaml"
	}

	isReloadEnv := true
	if _, err := os.Stat(yamlPath); err == nil {
		yamlFile, err := os.ReadFile(yamlPath)
		if err != nil {
			log.Println("failed to read yaml file, use config from env")
		}
		if err := yaml.Unmarshal(yamlFile, cfg); err != nil {
			log.Println("failed to unmarshal yaml file, use config from env")
		}
		isReloadEnv = false
	}

	// 2. load config from .env file
	if isReloadEnv {
		if err := env.Parse(cfg); err != nil {
			println("failed to parse .env file, use config from yaml")
		}
	}

	// 3. check if config is valid
	if cfg.AccessId == "" || cfg.AccessSecret == "" || cfg.Endpoint == "" {
		println("config is invalid, please check your config file or env")
		os.Exit(1)
	}

	return cfg
}
