package signal

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockContext struct {
	context.Context
	m *mock.Mock
}

func (c *mockContext) Done() <-chan struct{} {
	return c.m.Called().Get(0).(<-chan struct{})
}

func (c *mockContext) Err() error {
	return c.m.Called().Error(0)
}

func TestWaitForSignal(t *testing.T) {
	t.Run("it exits on context done", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		err := WaitForSignal(ctx)
		require.EqualError(t, err, context.Canceled.Error())
	})

	t.Run("it exits on interrupt signal", func(t *testing.T) {
		p, err := os.FindProcess(os.Getpid())
		require.NoError(t, err)

		done := make(chan struct{})
		defer close(done)

		m := &mock.Mock{}
		m.
			On("Done").Run(
			func(args mock.Arguments) {
				err = p.Signal(os.Interrupt)
				require.NoError(t, err)
			}).Return((<-chan struct{})(done)).Once().
			On("Err").Return(nil).Once()

		ctx := &mockContext{m: m}
		err = WaitForSignal(ctx)
		require.EqualError(t, err, os.Interrupt.String())
	})
}
