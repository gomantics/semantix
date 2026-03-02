package settings

import (
	"github.com/gomantics/semantix/internal/api/web"
	"github.com/gomantics/semantix/internal/domains/settings"
	"go.uber.org/zap"
)

type StatusResponse struct {
	SetupComplete    bool `json:"setup_complete"`
	OpenAIConfigured bool `json:"openai_configured"`
}

func Status(c web.Context) error {
	return c.OK(StatusResponse{
		SetupComplete:    settings.IsSetupComplete(),
		OpenAIConfigured: settings.IsOpenAIConfigured(),
	})
}

type SettingResponse struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	IsSecret bool   `json:"is_secret"`
	Updated  int64  `json:"updated"`
}

type GetResponse struct {
	Settings []SettingResponse `json:"settings"`
}

func Get(c web.Context) error {
	ctx := c.Request().Context()

	list, err := settings.List(ctx)
	if err != nil {
		c.L.Error("failed to list settings", zap.Error(err))
		return c.InternalError("failed to list settings")
	}

	resp := make([]SettingResponse, len(list))
	for i, s := range list {
		resp[i] = SettingResponse{
			Key:      s.Key,
			Value:    s.Value,
			IsSecret: s.IsSecret,
			Updated:  s.Updated,
		}
	}

	return c.OK(GetResponse{Settings: resp})
}

type UpdateRequest struct {
	OpenAIAPIKey string `json:"openai_api_key"`
}

func Update(c web.Context) error {
	var req UpdateRequest
	if err := c.Bind(&req); err != nil {
		return c.BadRequest("invalid request body")
	}

	if req.OpenAIAPIKey == "" {
		return c.BadRequest("openai_api_key is required")
	}

	ctx := c.Request().Context()

	if err := settings.SetOpenAIKey(ctx, req.OpenAIAPIKey); err != nil {
		c.L.Error("failed to save openai api key", zap.Error(err))
		return c.InternalError("failed to save openai api key")
	}

	return Get(c)
}
