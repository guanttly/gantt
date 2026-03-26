package intent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gantt-saas/internal/ai"

	"go.uber.org/zap"
)

const intentSystemPrompt = `You are a scheduling assistant. Extract the intent and entities from the user input.
Available intents: create_schedule, adjust_schedule, query_schedule, query_rule, chat
Available entities: date_range, department, shift_name, employee_name, rule_type
Return JSON only: {"action":"...","entities":{...},"confidence":0.0~1.0}`

// Intent represents a parsed intent.
type Intent struct {
	Action     string            `json:"action"`
	Entities   map[string]string `json:"entities"`
	Confidence float64           `json:"confidence"`
}

// Parser parses user input into intents.
type Parser struct {
	provider ai.Provider
	logger   *zap.Logger
}

// NewParser creates a new intent parser.
func NewParser(provider ai.Provider, logger *zap.Logger) *Parser {
	return &Parser{provider: provider, logger: logger.Named("intent")}
}

// Parse parses user input and returns the intent.
func (p *Parser) Parse(ctx context.Context, userInput string) (*Intent, error) {
	resp, err := p.provider.Chat(ctx, ai.ChatRequest{
		Messages: []ai.Message{
			{Role: "system", Content: intentSystemPrompt},
			{Role: "user", Content: userInput},
		},
		Temperature: 0.1,
	})
	if err != nil {
		return nil, fmt.Errorf("intent parse failed: %w", err)
	}

	intent := &Intent{Entities: make(map[string]string)}
	content := strings.TrimSpace(resp.Content)
	if idx := strings.Index(content, "{"); idx != -1 {
		if endIdx := strings.LastIndex(content, "}"); endIdx > idx {
			content = content[idx : endIdx+1]
		}
	}

	if err := json.Unmarshal([]byte(content), intent); err != nil {
		p.logger.Warn("intent parse JSON failed", zap.Error(err))
		return &Intent{Action: "chat", Entities: make(map[string]string), Confidence: 0.5}, nil
	}

	p.logger.Info("intent parsed", zap.String("action", intent.Action), zap.Float64("confidence", intent.Confidence))
	return intent, nil
}
