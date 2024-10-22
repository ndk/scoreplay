package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockHTTPHandler struct {
	m *mock.Mock
}

func (m *mockHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.m.Called(w, r)
}

func TestRecoveryMiddleware(t *testing.T) {
	t.Run("it moves on the call forward", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("ServeHTTP", http.ResponseWriter(nil), (*http.Request)(nil)).Return().Once()

		handler := RecoveryMiddleware(&mockHTTPHandler{m: m})
		handler.ServeHTTP(nil, nil)

		require.True(t, m.AssertExpectations(t))
	})

	t.Run("it returns HTTP error if the next handler paniced", func(t *testing.T) {
		m := &mock.Mock{}
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		m.On("ServeHTTP", w, r).Panic("test").Once()

		handler := RecoveryMiddleware(&mockHTTPHandler{m: m})
		handler.ServeHTTP(w, r)
		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.Equal(t, http.StatusText(http.StatusInternalServerError)+"\n", w.Body.String())
		require.True(t, m.AssertExpectations(t))
	})
}
