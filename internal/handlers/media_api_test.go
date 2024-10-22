package handlers

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"scoreplay/internal/service"
	"scoreplay/pkg/api"
)

type mockService struct {
	m *mock.Mock
}

func (m *mockService) CreateTag(ctx context.Context, params service.CreateTagParams) error {
	return m.m.Called(ctx, params).Error(0)
}

func (m *mockService) ListTags(ctx context.Context) (service.ListTagsResult, error) {
	args := m.m.Called(ctx)
	return args.Get(0).(service.ListTagsResult), args.Error(1)
}

func (m *mockService) ListMedia(ctx context.Context, params service.ListMediaParams) (service.ListMediaResult, error) {
	args := m.m.Called(ctx, params)
	return args.Get(0).(service.ListMediaResult), args.Error(1)
}

func (m *mockService) CreateMedia(ctx context.Context, params service.CreateMediaParams) (*service.CreateMediaResult, error) {
	args := m.m.Called(ctx, params)
	return args.Get(0).(*service.CreateMediaResult), args.Error(1)
}

func TestNewMediaAPI(t *testing.T) {
	ms := &mockService{}
	h := NewMediaAPI(ms)
	require.Equal(t, &handler{mediaService: ms}, h)
}

func TestHandler_GetTags(t *testing.T) {
	ctx := context.Background()

	t.Run("if fails if tags service fails", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("ListTags", ctx).Return((service.ListTagsResult)(nil), assert.AnError).Once()

		h := NewMediaAPI(&mockService{m: m})
		_, err := h.GetTags(ctx, api.GetTagsRequestObject{})

		require.EqualError(t, err, `listing tags: `+assert.AnError.Error())
		require.True(t, m.AssertExpectations(t))
	})

	t.Run("it returns tags", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("ListTags", ctx).Return(service.ListTagsResult{"tag1", "tag2"}, nil).Once()

		h := NewMediaAPI(&mockService{m: m})
		resp, err := h.GetTags(ctx, api.GetTagsRequestObject{})

		require.NoError(t, err)
		assert.Equal(t, api.GetTags200JSONResponse{"tag1", "tag2"}, resp)
		require.True(t, m.AssertExpectations(t))
	})
}

func TestHandler_PostTags(t *testing.T) {
	ctx := context.Background()

	t.Run("if fails if tags service fails", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("CreateTag", ctx, service.CreateTagParams{Name: "tag"}).Return(assert.AnError).Once()

		h := NewMediaAPI(&mockService{m: m})
		_, err := h.PostTags(ctx, api.PostTagsRequestObject{Body: &api.PostTagsJSONRequestBody{Name: "tag"}})

		require.EqualError(t, err, `creating tag: `+assert.AnError.Error())
		require.True(t, m.AssertExpectations(t))
	})

	t.Run("it returns success", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("CreateTag", ctx, service.CreateTagParams{Name: "tag"}).Return(nil).Once()

		h := NewMediaAPI(&mockService{m: m})
		resp, err := h.PostTags(ctx, api.PostTagsRequestObject{Body: &api.PostTagsJSONRequestBody{Name: "tag"}})

		require.NoError(t, err)
		assert.Equal(t, api.PostTags201Response{}, resp)
		require.True(t, m.AssertExpectations(t))
	})
}

func TestHandler_PostMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("if fails if upload service fails", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("CreateMedia", ctx, service.CreateMediaParams{Name: "name", Tags: []string{"tag1", "tag2"}}).Return((*service.CreateMediaResult)(nil), assert.AnError).Once()

		h := NewMediaAPI(&mockService{m: m})
		_, err := h.PostMedia(ctx, api.PostMediaRequestObject{Body: &api.PostMediaJSONRequestBody{Name: "name", Tags: []string{"tag1", "tag2"}}})

		require.EqualError(t, err, `creating upload: `+assert.AnError.Error())
		require.True(t, m.AssertExpectations(t))
	})

	t.Run("it returns success", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("CreateMedia", ctx, service.CreateMediaParams{Name: "name", Tags: []string{"tag1", "tag2"}}).
			Return(&service.CreateMediaResult{URL: "url", Method: http.MethodPut, SignedHeader: http.Header{"x-amz-meta-name": []string{"name"}, "x-amz-meta-tags": []string{"tag1", "tag2"}}}, nil).Once()

		h := NewMediaAPI(&mockService{m: m})
		resp, err := h.PostMedia(ctx, api.PostMediaRequestObject{Body: &api.PostMediaJSONRequestBody{Name: "name", Tags: []string{"tag1", "tag2"}}})

		require.NoError(t, err)
		assert.Equal(t, api.PostMedia201JSONResponse{Url: "url", Method: http.MethodPut, SignedHeader: http.Header{"x-amz-meta-name": []string{"name"}, "x-amz-meta-tags": []string{"tag1", "tag2"}}}, resp)
		require.True(t, m.AssertExpectations(t))
	})
}

func TestHandler_GetMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("if fails if query service fails", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("ListMedia", ctx, service.ListMediaParams{Tag: "tag1"}).Return((service.ListMediaResult)(nil), assert.AnError).Once()

		h := NewMediaAPI(&mockService{m: m})
		_, err := h.GetMedia(ctx, api.GetMediaRequestObject{Params: api.GetMediaParams{Tag: "tag1"}})

		require.EqualError(t, err, `listing media: `+assert.AnError.Error())
		require.True(t, m.AssertExpectations(t))
	})

	t.Run("it returns media", func(t *testing.T) {
		m := &mock.Mock{}
		m.On("ListMedia", ctx, service.ListMediaParams{Tag: "tag1"}).
			Return(service.ListMediaResult{{Name: "name1", Tags: []string{"tag1", "tag2"}}, {Name: "name2", Tags: []string{"tag2", "tag3"}}}, nil).Once()

		h := NewMediaAPI(&mockService{m: m})
		resp, err := h.GetMedia(ctx, api.GetMediaRequestObject{Params: api.GetMediaParams{Tag: "tag1"}})

		require.NoError(t, err)
		assert.Equal(t, api.GetMedia200JSONResponse{{Name: "name1", Tags: []string{"tag1", "tag2"}}, {Name: "name2", Tags: []string{"tag2", "tag3"}}}, resp)
		require.True(t, m.AssertExpectations(t))
	})
}
