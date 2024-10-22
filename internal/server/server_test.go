package server

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	ctx := context.Background()
	s := miniredis.RunT(t)
	defer s.Close()

	t.Run("it stops if the context is canceled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()

		var cfg Config
		cfg.Redis.InitAddress = []string{s.Addr()}
		cfg.Redis.DisableCache = true
		cfg.AWS.S3.UsePathStyle = true
		cfg.Server.Address = ":"
		err := Run(ctx, cfg)

		require.ErrorIs(t, err, context.Canceled)
		require.EqualError(t, err, "context canceled")
	})

	t.Run("it fails if the redis client cannot be created", func(t *testing.T) {
		var cfg Config
		err := Run(ctx, cfg)
		require.EqualError(t, err, "creating redis client: no alive address in InitAddress")
	})

	t.Run("it fails if it cannot parse AWS endpoint URL", func(t *testing.T) {
		var cfg Config
		cfg.Redis.InitAddress = []string{s.Addr()}
		cfg.Redis.DisableCache = true
		cfg.AWS.EndpointURL = "1:/1:-"
		cfg.Server.Address = ":"
		err := Run(ctx, cfg)
		require.EqualError(t, err, `parsing endpoint url: parse "1:/1:-": first path segment in URL cannot contain colon`)
	})

	t.Run("it fails if it cannot serve", func(t *testing.T) {
		var cfg Config
		cfg.Redis.InitAddress = []string{s.Addr()}
		cfg.Redis.DisableCache = true
		cfg.Server.Address = "1:/1:-"
		err := Run(ctx, cfg)
		require.EqualError(t, err, `listen tcp: address 1:/1:-: too many colons in address`)
	})
}
