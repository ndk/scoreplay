package service

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
	rmock "github.com/redis/rueidis/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func pT[T any](v T) *T {
	return &v
}

func parseURL(t *testing.T, s string) url.URL {
	t.Helper()
	u, err := url.Parse(s)
	require.NoError(t, err)
	return *u
}

type mockPresignClient struct {
	m *mock.Mock
}

func (m *mockPresignClient) PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	args := m.m.Called(ctx, params, optFns)
	return args.Get(0).(*v4.PresignedHTTPRequest), args.Error(1)
}

func mockUUID(m *mock.Mock) func() (uuid.UUID, error) {
	return func() (uuid.UUID, error) {
		args := m.MethodCalled("generateUUID")
		return args.Get(0).(uuid.UUID), args.Error(1)
	}
}

func TestNewMediaService(t *testing.T) {
	rc := rmock.NewClient(nil)
	pc := &mockPresignClient{}
	endpointURL := url.URL{}
	bucket := "bucket"

	s := NewMediaService(rc, pc, endpointURL, bucket)

	require.NotNil(t, s)
	require.Equal(t, bucket, s.bucket)
	require.Equal(t, endpointURL, s.endpointURL)
	require.NotNil(t, s.generateUUID)
	require.Equal(t, pc, s.presignClient)
	require.Equal(t, rc, s.rueidisClient)
}

func TestMediaService_CreateTag(t *testing.T) {
	ctx := context.Background()

	t.Run("it fails if redis fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SADD", tagsKey, "mytag")).Return(rmock.ErrorResult(assert.AnError))

		s := NewMediaService(rc, nil, url.URL{}, "")
		err := s.CreateTag(ctx, CreateTagParams{Name: "mytag"})

		require.EqualError(t, err, "creating tag: "+assert.AnError.Error())
		require.True(t, ctrl.Satisfied())
	})

	t.Run("happy path", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SADD", tagsKey, "mytag")).Return(rmock.ErrorResult(nil))

		s := NewMediaService(rc, nil, url.URL{}, "")
		err := s.CreateTag(ctx, CreateTagParams{Name: "mytag"})

		require.NoError(t, err)
		require.True(t, ctrl.Satisfied())
	})
}

func TestMediaService_ListTags(t *testing.T) {
	ctx := context.Background()

	t.Run("it fails if redis fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsKey)).Return(rmock.ErrorResult(assert.AnError))

		s := NewMediaService(rc, nil, url.URL{}, "")
		tags, err := s.ListTags(ctx)

		require.Nil(t, tags)
		require.EqualError(t, err, "getting tags from redis: "+assert.AnError.Error())
		require.True(t, ctrl.Satisfied())
	})

	t.Run("happy path", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsKey)).
			Return(rmock.Result(rmock.RedisArray(rmock.RedisString("tag1"), rmock.RedisString("tag2"))))

		s := NewMediaService(rc, nil, url.URL{}, "")
		tags, err := s.ListTags(ctx)

		require.NoError(t, err)
		require.Equal(t, ListTagsResult{"tag1", "tag2"}, tags)
		require.True(t, ctrl.Satisfied())
	})
}

func TestMediaService_ListMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("it fails if redis fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsPrefix+"mytag")).Return(rmock.ErrorResult(assert.AnError))

		s := NewMediaService(rc, nil, url.URL{}, "")
		media, err := s.ListMedia(ctx, ListMediaParams{Tag: "mytag"})

		require.Nil(t, media)
		require.EqualError(t, err, "getting media keys from redis: "+assert.AnError.Error())
		require.True(t, ctrl.Satisfied())
	})

	t.Run("it fails if getting media fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsPrefix+"mytag")).
			Return(rmock.Result(rmock.RedisArray(rmock.RedisString("key1"), rmock.RedisString("key2"))))
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HGETALL", mediaPrefix+"key1"),
			rmock.Match("HGETALL", mediaPrefix+"key2"),
		).Return([]rueidis.RedisResult{
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name1"),
				tagsField: rmock.RedisString("tag1,tag2"),
			})),
			rmock.ErrorResult(assert.AnError),
		})

		s := NewMediaService(rc, nil, parseURL(t, "http://test"), "mybucket")
		media, err := s.ListMedia(ctx, ListMediaParams{Tag: "mytag"})

		require.Nil(t, media)
		require.EqualError(t, err, "getting media record 1: "+assert.AnError.Error())
		require.True(t, ctrl.Satisfied())
	})

	t.Run("it fails if decoding media fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsPrefix+"mytag")).
			Return(rmock.Result(rmock.RedisArray(rmock.RedisString("key1"), rmock.RedisString("key2"))))
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HGETALL", mediaPrefix+"key1"),
			rmock.Match("HGETALL", mediaPrefix+"key2"),
		).Return([]rueidis.RedisResult{
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name1"),
				tagsField: rmock.RedisString("tag1,tag2"),
			})),
			rmock.Result(rmock.RedisString("name2")),
		})

		s := NewMediaService(rc, nil, parseURL(t, "http://test"), "mybucket")
		media, err := s.ListMedia(ctx, ListMediaParams{Tag: "mytag"})

		require.Nil(t, media)
		require.EqualError(t, err, `decoding media record 1: rueidis: parse error: redis message type simple string is not a map/array/set or its length is not even`)
		require.True(t, ctrl.Satisfied())
	})

	t.Run("it fails if decoding tags fails", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsPrefix+"mytag")).
			Return(rmock.Result(rmock.RedisArray(rmock.RedisString("key1"), rmock.RedisString("key2"))))
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HGETALL", mediaPrefix+"key1"),
			rmock.Match("HGETALL", mediaPrefix+"key2"),
		).Return([]rueidis.RedisResult{
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name1"),
				tagsField: rmock.RedisString("tag1,tag2"),
			})),
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name2"),
				tagsField: rmock.RedisString("tag2,tag3%"),
			})),
		})

		s := NewMediaService(rc, nil, parseURL(t, "http://test"), "mybucket")
		media, err := s.ListMedia(ctx, ListMediaParams{Tag: "mytag"})

		require.Nil(t, media)
		require.EqualError(t, err, `decoding tag 1: invalid URL escape "%"`)
		require.True(t, ctrl.Satisfied())
	})

	t.Run("happy path", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().Do(ctx, rmock.Match("SMEMBERS", tagsPrefix+"mytag")).
			Return(rmock.Result(rmock.RedisArray(rmock.RedisString("key1"), rmock.RedisString("key2"))))
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HGETALL", mediaPrefix+"key1"),
			rmock.Match("HGETALL", mediaPrefix+"key2"),
		).Return([]rueidis.RedisResult{
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name1"),
				tagsField: rmock.RedisString("tag1,tag2"),
			})),
			rmock.Result(rmock.RedisMap(map[string]rueidis.RedisMessage{
				nameField: rmock.RedisString("name2"),
				tagsField: rmock.RedisString("tag2,ta%2Cg3"),
			})),
		})

		s := NewMediaService(rc, nil, parseURL(t, "http://test"), "mybucket")
		media, err := s.ListMedia(ctx, ListMediaParams{Tag: "mytag"})

		require.NoError(t, err)
		require.Equal(t, ListMediaResult{
			{Key: "key1", Name: "name1", URL: parseURL(t, "http://test/mybucket/key1"), Tags: []string{"tag1", "tag2"}},
			{Key: "key2", Name: "name2", URL: parseURL(t, "http://test/mybucket/key2"), Tags: []string{"tag2", "ta,g3"}},
		}, media)
		require.True(t, ctrl.Satisfied())
	})
}

func TestMediaService_CreateMedia(t *testing.T) {
	ctx := context.Background()

	t.Run("it fails if generating UUID fails", func(t *testing.T) {
		m := &mock.Mock{}
		s := NewMediaService(nil, nil, url.URL{}, "")
		s.generateUUID = mockUUID(m)
		m.On("generateUUID").Return(uuid.UUID{}, assert.AnError).Once()

		result, err := s.CreateMedia(ctx, CreateMediaParams{})

		require.Nil(t, result)
		require.EqualError(t, err, "generating UUID: "+assert.AnError.Error())
	})

	t.Run("it fails if presigning fails", func(t *testing.T) {
		id, err := uuid.NewV7()
		require.NoError(t, err)

		m := &mock.Mock{}
		m.On("generateUUID").Return(id, nil).Once().
			On("PresignPutObject", ctx,
				&s3.PutObjectInput{
					Bucket: pT("bucket"),
					Key:    pT(id.String()),
				}, ([]func(*s3.PresignOptions))(nil)).
			Return((*v4.PresignedHTTPRequest)(nil), assert.AnError).Once()
		pc := &mockPresignClient{m: m}

		s := NewMediaService(nil, pc, url.URL{}, "bucket")
		s.generateUUID = mockUUID(m)
		result, err := s.CreateMedia(ctx, CreateMediaParams{})

		require.Nil(t, result)
		require.EqualError(t, err, "presigning put object: "+assert.AnError.Error())
	})

	t.Run("it fails if redis fails", func(t *testing.T) {
		id, err := uuid.NewV7()
		require.NoError(t, err)

		m := &mock.Mock{}
		m.On("generateUUID").Return(id, nil).Once().
			On("PresignPutObject", ctx,
				&s3.PutObjectInput{
					Bucket: pT("bucket"),
					Key:    pT(id.String()),
				}, ([]func(*s3.PresignOptions))(nil)).
			Return(&v4.PresignedHTTPRequest{
				URL:          "http://test/mybucket/key1",
				Method:       http.MethodPut,
				SignedHeader: http.Header{"X-Amz-Security-Token": []string{"token"}},
			}, nil).Once()
		pc := &mockPresignClient{m: m}

		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HSET", mediaPrefix+id.String(), nameField, "name1", tagsField, "tag1,tag2"),
			rmock.Match("SADD", tagsKey, "tag1"),
			rmock.Match("SADD", tagsPrefix+"tag1", id.String()),
			rmock.Match("SADD", tagsKey, "tag2"),
			rmock.Match("SADD", tagsPrefix+"tag2", id.String()),
		).Return([]rueidis.RedisResult{
			rmock.ErrorResult(assert.AnError),
		})

		s := NewMediaService(rc, pc, url.URL{}, "bucket")
		s.generateUUID = mockUUID(m)
		result, err := s.CreateMedia(ctx, CreateMediaParams{Name: "name1", Tags: []string{"tag1", "tag2"}})

		require.Nil(t, result)
		require.EqualError(t, err, "executing command 0: "+assert.AnError.Error())
		require.True(t, ctrl.Satisfied())
	})

	t.Run("happy path", func(t *testing.T) {
		id, err := uuid.NewV7()
		require.NoError(t, err)

		m := &mock.Mock{}
		m.On("generateUUID").Return(id, nil).Once().
			On("PresignPutObject", ctx,
				&s3.PutObjectInput{
					Bucket: pT("bucket"),
					Key:    pT(id.String()),
				}, ([]func(*s3.PresignOptions))(nil)).
			Return(&v4.PresignedHTTPRequest{
				URL:          "http://test/mybucket/key1",
				Method:       http.MethodPut,
				SignedHeader: http.Header{"X-Amz-Security-Token": []string{"token"}},
			}, nil).Once()
		pc := &mockPresignClient{m: m}

		ctrl := gomock.NewController(t)
		rc := rmock.NewClient(ctrl)
		rc.EXPECT().DoMulti(ctx,
			rmock.Match("HSET", mediaPrefix+id.String(), nameField, "name1", tagsField, "tag1,tag2"),
			rmock.Match("SADD", tagsKey, "tag1"),
			rmock.Match("SADD", tagsPrefix+"tag1", id.String()),
			rmock.Match("SADD", tagsKey, "tag2"),
			rmock.Match("SADD", tagsPrefix+"tag2", id.String()),
		).Return([]rueidis.RedisResult{
			rmock.Result(rmock.RedisInt64(1)),
			rmock.Result(rmock.RedisInt64(1)),
			rmock.Result(rmock.RedisInt64(1)),
			rmock.Result(rmock.RedisInt64(1)),
			rmock.Result(rmock.RedisInt64(1)),
		})

		s := NewMediaService(rc, pc, url.URL{}, "bucket")
		s.generateUUID = mockUUID(m)
		result, err := s.CreateMedia(ctx, CreateMediaParams{Name: "name1", Tags: []string{"tag1", "tag2"}})

		require.NoError(t, err)
		require.Equal(t, &v4.PresignedHTTPRequest{
			URL:          "http://test/mybucket/key1",
			Method:       http.MethodPut,
			SignedHeader: http.Header{"X-Amz-Security-Token": []string{"token"}},
		}, result)
		require.True(t, ctrl.Satisfied())
	})
}
