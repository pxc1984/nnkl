package extractor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pxc1984/nnkl-backend/store/models"
	"github.com/pxc1984/nnkl-backend/utils"
)

// YandexLLMClient — минимальный OpenAI-compatible клиент для Yandex LLM.
type YandexLLMClient struct {
	apiKey   string
	modelURI string
	host     string
	client   *http.Client
}

// NewYandexLLMClient создаёт клиент из переменных окружения.
func NewYandexLLMClient() *YandexLLMClient {
	modelURI := utils.Settings.YandexLLMModel
	if modelURI == "" && utils.Settings.YandexFolderID != "" {
		modelURI = fmt.Sprintf("gpt://%s/yandexgpt-lite/latest", utils.Settings.YandexFolderID)
	}
	return &YandexLLMClient{
		apiKey:   utils.Settings.YandexLLMAPIKey,
		modelURI: modelURI,
		host:     utils.Settings.YandexLLMHost,
		client:   &http.Client{Timeout: 120 * time.Second},
	}
}

// IsConfigured возвращает true, если клиент готов к использованию.
func (c *YandexLLMClient) IsConfigured() bool {
	return c.apiKey != "" && c.modelURI != "" && c.host != ""
}

type llmMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type llmRequest struct {
	Model    string       `json:"model"`
	Messages []llmMessage `json:"messages"`
}

type llmResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// Complete отправляет промпт к LLM и возвращает текст ответа.
func (c *YandexLLMClient) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	body, err := json.Marshal(llmRequest{
		Model: c.modelURI,
		Messages: []llmMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	})
	if err != nil {
		return "", fmt.Errorf("marshal llm request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create llm request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Api-Key "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("send llm request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read llm response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("llm returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var result llmResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("decode llm response: %w", err)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices in llm response")
	}
	return result.Choices[0].Message.Content, nil
}

// NumericFactExtractor извлекает числовые факты из текста с помощью LLM.
type NumericFactExtractor struct {
	llm *YandexLLMClient
}

// NewNumericFactExtractor создаёт новый экстрактор.
func NewNumericFactExtractor() *NumericFactExtractor {
	return &NumericFactExtractor{llm: NewYandexLLMClient()}
}

// IsConfigured возвращает true, если экстрактор готов к работе.
func (e *NumericFactExtractor) IsConfigured() bool {
	return e.llm.IsConfigured()
}

const numericExtractionSystemPrompt = `Ты — технический ассистент для горно-металлургической отрасли. Извлеки из предоставленного текста все числовые факты, относящиеся к материалам, процессам, оборудованию, параметрам и свойствам.

Для каждого факта верни JSON-объект со следующими полями:
- entityName: название сущности (например, "никелевая руда", "электролит", "ванна электроэкстракции")
- property: название свойства (например, "концентрация сульфатов", "температура плавления", "скорость потока", "pH", "плотность тока")
- value: числовое значение
- value2: второе значение для диапазонов (если указано "от X до Y", иначе 0)
- unit: единица измерения (например, "мг/л", "°C", "м/с", "А/м²")
- operator: один из "=", "<", "<=", ">", ">=", "between"
- rawText: точная цитата из текста, содержащая этот факт

Верни строго JSON-массив. Если фактов нет — верни пустой массив []. Не добавляй пояснений вне JSON.`

// Extract извлекает числовые факты из chunkText.
func (e *NumericFactExtractor) Extract(ctx context.Context, chunkText, documentID, chunkID string) ([]models.NumericFact, error) {
	if !e.IsConfigured() {
		return nil, nil
	}

	userPrompt := fmt.Sprintf("Текст:\n\n%s\n\nИзвлеки числовые факты в формате JSON-массива.", chunkText)
	raw, err := e.llm.Complete(ctx, numericExtractionSystemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm completion: %w", err)
	}

	raw = extractJSON(raw)
	var facts []models.NumericFact
	if err := json.Unmarshal([]byte(raw), &facts); err != nil {
		return nil, fmt.Errorf("unmarshal facts: %w\nresponse: %s", err, raw)
	}

	now := time.Now().UTC()
	for i := range facts {
		facts[i].ID = uuid.NewString()
		facts[i].DocumentID = documentID
		facts[i].ChunkID = chunkID
		facts[i].CreatedAt = now
		facts[i].Property = strings.ToLower(strings.TrimSpace(facts[i].Property))
		facts[i].Unit = strings.ToLower(strings.TrimSpace(facts[i].Unit))
		facts[i].Operator = normalizeOperator(facts[i].Operator)
	}

	return facts, nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimPrefix(s, "```")
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	return s
}

func normalizeOperator(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "≤", "<=", "=<", "не более", "не больше", "до", "меньше или равно":
		return "<="
	case "<", "меньше":
		return "<"
	case "≥", ">=", "=>", "не менее", "не меньше", "больше или равно":
		return ">="
	case ">", "больше":
		return ">"
	case "between", "от", "диапазон":
		return "between"
	default:
		return "="
	}
}
