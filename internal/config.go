package internal

import "log/slog"

type Config struct {
	Env        string           `mapstructure:"env"`
	Log        LogConfig        `mapstructure:"log"`
	HTTPServer HTTPServerConfig `mapstructure:"http_server"`
	Database   DatabaseConfig   `mapstructure:"database"`
	JWT        JWTConfig        `mapstructure:"jwt"`
}

type LogConfig struct {
	Level string `mapstructure:"level"`
}

func (c LogConfig) ParseSlogLevel() slog.Level {
	switch c.Level {
	case "DEBUG":
		return slog.LevelDebug
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type HTTPServerConfig struct {
	Port string `mapstructure:"port"`
}

type DatabaseConfig struct {
	Source string `mapstructure:"source"`
}

type JWTConfig struct {
	Secret string `mapstructure:"secret"`
}
