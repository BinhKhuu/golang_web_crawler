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
	Model        = "gemma4"
	MaxMemoryMBs = 16384
)

var ErrNoJson = errors.New("no JSON block found in LLM response")

type LLMService struct {
	ModelName    string
	maxMemoryMBs int
	Client       *api.Client
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

func (l *LLMService) QueryLLM(prompt string) ([]models.ExtractedJobData, error) {
	req := &api.GenerateRequest{
		Model:  l.ModelName,
		Prompt: prompt,
		Options: map[string]any{
			"num_ctx": MaxMemoryMBs, // This is temporary for THIS specific call only
		},
		Stream: new(bool), // Set to false for a single complete response
	}

	var fullResponse strings.Builder

	err := l.Client.Generate(context.Background(), req, func(resp api.GenerateResponse) error {
		fullResponse.WriteString(resp.Response)
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 1. Extract the JSON block using Regex
	re := regexp.MustCompile("(?s)```json\n?(.*?)\n?```")
	message := fullResponse.String()
	match := re.FindStringSubmatch(message)
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
	} else {
		return nil, ErrNoJson
	}
}
