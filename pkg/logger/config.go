package logger

import (
	"github.com/elskow/go-microservice-template/config"
)

type Config struct {
	EnableStdout   bool
	EnableOTLP     bool
	BufferSize     int
	DropOnFull     bool
	OTLPEndpoint   string
	ServiceName    string
	ServiceVersion string
	Environment    string
}

func LoadConfig(serviceName, serviceVersion string) Config {
	cfg := config.Get()
	return Config{
		EnableStdout:   cfg.EnableStdoutLogs,
		EnableOTLP:     cfg.EnableOTLPLogs,
		BufferSize:     cfg.LogBufferSize,
		DropOnFull:     cfg.LogDropOnFull,
		OTLPEndpoint:   cfg.OTELExporterEndpoint,
		ServiceName:    serviceName,
		ServiceVersion: serviceVersion,
		Environment:    getEnvironment(cfg.AppEnv),
	}
}

func getEnvironment(env string) string {
	if env == "" {
		return "development"
	}
	return env
}
