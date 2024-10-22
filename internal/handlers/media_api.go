package handlers

import (
	"context"
	"fmt"

	"scoreplay/internal/service"
	"scoreplay/pkg/api"
)

type mediaService interface {
	CreateTag(ctx context.Context, params service.CreateTagParams) error
	ListTags(ctx context.Context) (service.ListTagsResult, error)
	ListMedia(ctx context.Context, params service.ListMediaParams) (service.ListMediaResult, error)
	CreateMedia(ctx context.Context, params service.CreateMediaParams) (*service.CreateMediaResult, error)
}

type handler struct {
	mediaService mediaService
}

func NewMediaAPI(mediaService mediaService) *handler {
	return &handler{mediaService: mediaService}
}

func (h handler) GetTags(ctx context.Context, request api.GetTagsRequestObject) (api.GetTagsResponseObject, error) {
	tags, err := h.mediaService.ListTags(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing tags: %w", err)
	}

	return (api.GetTags200JSONResponse)(tags), nil
}

func (h handler) PostTags(ctx context.Context, request api.PostTagsRequestObject) (api.PostTagsResponseObject, error) {
	if err := h.mediaService.CreateTag(ctx, service.CreateTagParams{Name: request.Body.Name}); err != nil {
		return nil, fmt.Errorf("creating tag: %w", err)
	}

	return api.PostTags201Response{}, nil
}

func (h handler) PostMedia(ctx context.Context, request api.PostMediaRequestObject) (api.PostMediaResponseObject, error) {
	ur, err := h.mediaService.CreateMedia(ctx,
		service.CreateMediaParams{
			Name: request.Body.Name,
			Tags: request.Body.Tags,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("creating upload: %w", err)
	}

	return api.PostMedia201JSONResponse{
		Method:       ur.Method,
		SignedHeader: ur.SignedHeader,
		Url:          ur.URL,
	}, nil
}

func (h handler) GetMedia(ctx context.Context, request api.GetMediaRequestObject) (api.GetMediaResponseObject, error) {
	media, err := h.mediaService.ListMedia(ctx, service.ListMediaParams{Tag: request.Params.Tag})
	if err != nil {
		return nil, fmt.Errorf("listing media: %w", err)
	}

	response := make(api.GetMedia200JSONResponse, len(media))
	for i, m := range media {
		response[i] = api.Media{
			Name: m.Name,
			Url:  m.URL.String(),
			Tags: m.Tags,
		}
	}

	return response, nil
}
