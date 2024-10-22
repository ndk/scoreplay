package service

import (
	"context"
	"fmt"
	"net/url"
	"path"
	"strings"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/redis/rueidis"
)

const (
	tagsKey     = "tags"
	tagsPrefix  = tagsKey + ":"
	mediaPrefix = "media:"
	nameField   = "name"
	tagsField   = "tags"
)

type presignClient interface {
	PresignPutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error)
}

type mediaService struct {
	bucket        string
	endpointURL   url.URL
	generateUUID  func() (uuid.UUID, error)
	presignClient presignClient
	rueidisClient rueidis.Client
}

func NewMediaService(rueidisClient rueidis.Client, presignClient presignClient, endpointURL url.URL, bucket string) *mediaService {
	return &mediaService{
		bucket:        bucket,
		endpointURL:   endpointURL,
		generateUUID:  uuid.NewV7,
		presignClient: presignClient,
		rueidisClient: rueidisClient,
	}
}

type CreateTagParams struct {
	Name string
}

func (s mediaService) CreateTag(ctx context.Context, params CreateTagParams) error {
	if err := s.rueidisClient.Do(ctx, s.rueidisClient.B().Sadd().Key(tagsKey).Member(params.Name).Build()).Error(); err != nil {
		return fmt.Errorf("creating tag: %w", err)
	}

	return nil
}

type Tag = string
type ListTagsResult []Tag

func (s mediaService) ListTags(ctx context.Context) (ListTagsResult, error) {
	tags, err := s.rueidisClient.Do(ctx, s.rueidisClient.B().Smembers().Key(tagsKey).Build()).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("getting tags from redis: %w", err)
	}

	return tags, nil
}

type ListMediaParams struct {
	Tag string
}
type MediaRecord struct {
	Key  string
	Name string
	URL  url.URL
	Tags []string
}
type ListMediaResult []MediaRecord

func (s mediaService) ListMedia(ctx context.Context, params ListMediaParams) (ListMediaResult, error) {
	keys, err := s.rueidisClient.Do(ctx, s.rueidisClient.B().Smembers().Key(tagsPrefix+params.Tag).Build()).AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("getting media keys from redis: %w", err)
	}

	cmds := make(rueidis.Commands, len(keys))
	result := make(ListMediaResult, len(cmds))
	for i, key := range keys {
		cmds[i] = s.rueidisClient.B().Hgetall().Key(mediaPrefix + key).Build()
	}
	for i, resp := range s.rueidisClient.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil {
			return nil, fmt.Errorf("getting media record %d: %w", i, err)
		}

		record, err := resp.AsStrMap()
		if err != nil {
			return nil, fmt.Errorf("decoding media record %d: %w", i, err)
		}

		encodedTags := strings.Split(record[tagsField], ",")
		tags := make([]string, len(encodedTags))
		for i, encodedTag := range encodedTags {
			tag, err := url.QueryUnescape(encodedTag)
			if err != nil {
				return nil, fmt.Errorf("decoding tag %d: %w", i, err)
			}
			tags[i] = tag
		}

		result[i] = MediaRecord{
			Key:  keys[i],
			Name: record[nameField],
			Tags: tags,
			URL:  s.endpointURL,
		}
		result[i].URL.Path = path.Join("/", result[i].URL.Path, s.bucket, keys[i])
	}

	return result, nil
}

type CreateMediaParams struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}
type CreateMediaResult = v4.PresignedHTTPRequest

func (s mediaService) CreateMedia(ctx context.Context, params CreateMediaParams) (*CreateMediaResult, error) {
	key, err := s.generateUUID()
	if err != nil {
		return nil, fmt.Errorf("generating UUID: %w", err)
	}

	keyStr := key.String()
	request, err := s.presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &s.bucket,
		Key:    &keyStr,
	})
	if err != nil {
		return nil, fmt.Errorf("presigning put object: %w", err)
	}

	encodedTags := make([]string, len(params.Tags))
	for i, tag := range params.Tags {
		encodedTags[i] = url.QueryEscape(tag)
	}

	cmds := make(rueidis.Commands, 0, len(params.Tags)*2+1)
	cmds = append(cmds,
		s.rueidisClient.B().Hset().Key(mediaPrefix+keyStr).FieldValue().
			FieldValue(nameField, params.Name).
			FieldValue(tagsField, strings.Join(encodedTags, ",")).
			Build(),
	)
	for _, tag := range params.Tags {
		cmds = append(cmds,
			s.rueidisClient.B().Sadd().Key(tagsKey).Member(tag).Build(),
			s.rueidisClient.B().Sadd().Key(tagsPrefix+tag).Member(keyStr).Build(),
		)
	}
	for i, resp := range s.rueidisClient.DoMulti(ctx, cmds...) {
		if err := resp.Error(); err != nil {
			return nil, fmt.Errorf("executing command %d: %w", i, err)
		}
	}

	return request, nil
}
