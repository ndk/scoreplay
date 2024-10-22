package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
	"go-simpler.org/env"

	"scoreplay/internal/logger"
	"scoreplay/internal/server"
)

var (
	helpFlag = flag.Bool("h", false, "Show help message") //nolint: gochecknoglobals
	opt      = &env.Options{NameSep: "_", SliceSep: ","}  //nolint: gochecknoglobals
)

func run(ctx context.Context) error {
	var cfg server.Config
	if *helpFlag {
		fmt.Println("Usage:")
		env.Usage(&server.Config{}, os.Stdout, opt)
		return nil
	}

	if err := env.Load(&cfg, opt); err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	ctx = logger.NewLogger(log.Logger, cfg.Logger).WithContext(ctx)
	log.Ctx(ctx).Info().Interface("cfg", cfg).Msg("config loaded")

	if err := server.Run(ctx, cfg); err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}

	return nil
}

func main() {
	flag.Parse()

	ctx := log.Logger.WithContext(context.Background())
	if err := run(ctx); err != nil {
		log.Ctx(ctx).Fatal().Err(err).Send()
	}
}
