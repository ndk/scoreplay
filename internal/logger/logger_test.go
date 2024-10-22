package logger

import (
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	l := NewLogger(zerolog.Logger{}, Config{Pretty: true, Level: "trace", Caller: true, Timestamp: true})
	require.NotNil(t, l)
}
