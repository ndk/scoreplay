//go:build integration

package tests

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go-simpler.org/env"

	"scoreplay/pkg/api"
)

func TestBasic(t *testing.T) {
	var cfg struct {
		AWS struct {
			EndpointUrl string `env:"ENDPOINT_URL,required"`
		} `env:"AWS"`
		Storage struct {
			Bucket string `env:"BUCKET,required"`
		} `env:"STORAGE"`
		APIURL string `env:"APIURL,required"`
	}
	err := env.Load(&cfg, &env.Options{NameSep: "_", SliceSep: ","})
	require.NoError(t, err)

	c, err := api.NewClientWithResponses(cfg.APIURL)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	expectTags := func(tags []api.Tag) {
		resp, err := c.GetTagsWithResponse(ctx)
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		require.Equal(t, &tags, resp.JSON200)
	}
	expectMedia := func(tag api.Tag, media []api.Media) {
		resp, err := c.GetMediaWithResponse(ctx, &api.GetMediaParams{Tag: tag})
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, resp.StatusCode())
		for i, m := range *resp.JSON200 {
			require.Equal(t, media[i].Name, m.Name, i)
			require.Equal(t, media[i].Tags, m.Tags, i)
			require.Contains(t, m.Url, cfg.AWS.EndpointUrl+"/"+cfg.Storage.Bucket+"/")
		}
	}
	addTag := func(name string) {
		resp, err := c.PostTagsWithResponse(ctx, api.PostTagsJSONRequestBody{Name: name})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode())
	}
	addMedia := func(tags []api.Tag, name string) {
		resp, err := c.PostMediaWithResponse(ctx, api.PostMediaJSONRequestBody{Tags: tags, Name: name})
		require.NoError(t, err)
		require.Equal(t, http.StatusCreated, resp.StatusCode())
		require.Equal(t, http.MethodPut, resp.JSON201.Method)
		require.Equal(t, http.Header{"Host": []string{"localstack:4566"}}, resp.JSON201.SignedHeader)
		require.Contains(t, resp.JSON201.Url, cfg.AWS.EndpointUrl+"/"+cfg.Storage.Bucket+"/")
	}

	expectTags([]api.Tag{})
	addTag("tag1")
	expectTags([]api.Tag{"tag1"})
	addTag("tag2")
	expectTags([]api.Tag{"tag1", "tag2"})
	addTag("tag1")
	expectTags([]api.Tag{"tag1", "tag2"})
	expectMedia("tag1", []api.Media{})
	addMedia([]api.Tag{"tag1", "tag2"}, "media1")
	expectMedia("tag1", []api.Media{{Name: "media1", Tags: []api.Tag{"tag1", "tag2"}}})
	addMedia([]api.Tag{"tag2", "ta,g3"}, "media2")
	expectMedia("tag1", []api.Media{
		{Name: "media1", Tags: []api.Tag{"tag1", "tag2"}},
	})
	expectMedia("tag2", []api.Media{
		{Name: "media1", Tags: []api.Tag{"tag1", "tag2"}},
		{Name: "media2", Tags: []api.Tag{"tag2", "ta,g3"}},
	})
	expectMedia("ta,g3", []api.Media{
		{Name: "media2", Tags: []api.Tag{"tag2", "ta,g3"}},
	})
}
