package llm

import (
	"context"
	"encoding/json"
	"errors"
	"golangwebcrawler/internal/models"
	"regexp"
	"strings"

	"github.com/ollama/ollama/api"
)

const (
	Model        = "gemma4:e4b-mlx"
	MaxMemoryMBs = 16384
)

var ErrNoJson = errors.New("no JSON block found in LLM response")

type generator interface {
	Generate(ctx context.Context, req *api.GenerateRequest, fn api.GenerateResponseFunc) error
}

type LLMService struct {
	ModelName    string
	maxMemoryMBs int
	Client       generator
}

func NewLLMService() (*LLMService, error) {
	client, err := initLLMConnection()
	if err != nil {
		return nil, err
	}

	return &LLMService{
		ModelName:    Model,
		maxMemoryMBs: MaxMemoryMBs,
		Client:       client,
	}, nil
}

func initLLMConnection() (*api.Client, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		return nil, err
	}

	return client, nil
}

func extractJSONFromResponse(response string) ([]models.ExtractedJobData, error) {
	re := regexp.MustCompile("(?s)```json\n?(.*?)\n?```")
	match := re.FindStringSubmatch(response)
	if len(match) > 1 {
		jsonStr := match[1]
		raw := strings.TrimSpace(jsonStr)
		raw = strings.TrimPrefix(raw, "```json")
		raw = strings.TrimPrefix(raw, "```")
		raw = strings.TrimSuffix(raw, "```")
		raw = strings.TrimSpace(raw)

		if raw == "" {
			return []models.ExtractedJobData{}, nil
		}
		var job []models.ExtractedJobData
		if err := json.Unmarshal([]byte(raw), &job); err != nil {
			return nil, err
		}

		if len(job) == 0 {
			return []models.ExtractedJobData{}, ErrNoJson
		}

		return job, nil
	}

	return nil, ErrNoJson
}

func (l *LLMService) QueryLLM(ctx context.Context, prompt string) ([]models.ExtractedJobData, error) {
	req := &api.GenerateRequest{
		Model:  l.ModelName,
		Prompt: prompt,
		Options: map[string]any{
			"num_ctx": MaxMemoryMBs,
		},
		Stream: new(bool),
	}

	var fullResponse strings.Builder

	err := l.Client.Generate(ctx, req, func(resp api.GenerateResponse) error {
		fullResponse.WriteString(resp.Response)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return extractJSONFromResponse(fullResponse.String())
}
