package logger

import (
	"github.com/rs/zerolog"
)

type Config struct {
	Pretty    bool   `env:"PRETTY" default:"false"`
	Level     string `env:"LEVEL" default:"info"`
	Caller    bool   `env:"CALLER" default:"false"`
	Timestamp bool   `env:"TIMESTAMP" default:"false"`
}

func NewLogger(l zerolog.Logger, cfg Config) zerolog.Logger {
	if cfg.Pretty {
		l = l.Output(zerolog.NewConsoleWriter())
	}
	if level, err := zerolog.ParseLevel(cfg.Level); err == nil {
		l = l.Level(level)
	}
	if cfg.Caller {
		l = l.With().Caller().Logger()
	}
	if cfg.Timestamp {
		l = l.With().Timestamp().Logger()
	}

	return l
}
