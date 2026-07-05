package data

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/pxc1984/nnkl-backend/store/models"
)

// NumericConstraint описывает числовое ограничение, найденное в вопросе пользователя.
type NumericConstraint struct {
	Property string
	Operator string  // "<=", "<", ">=", ">", "between"
	Value    float64 // для between — нижняя граница
	Value2   float64 // для between — верхняя граница
	Unit     string
}

// String формирует человекочитаемое описание ограничения для промпта.
func (c NumericConstraint) String() string {
	switch c.Operator {
	case "between":
		return fmt.Sprintf("%s от %.4g до %.4g %s", c.Property, c.Value, c.Value2, c.Unit)
	case "<=":
		return fmt.Sprintf("%s не более %.4g %s", c.Property, c.Value, c.Unit)
	case "<":
		return fmt.Sprintf("%s меньше %.4g %s", c.Property, c.Value, c.Unit)
	case ">=":
		return fmt.Sprintf("%s не менее %.4g %s", c.Property, c.Value, c.Unit)
	case ">":
		return fmt.Sprintf("%s больше %.4g %s", c.Property, c.Value, c.Unit)
	default:
		return fmt.Sprintf("%s %.4g %s", c.Property, c.Value, c.Unit)
	}
}

var (
	// property operator value [unit]: "сульфаты ≤300 мг/л", "pH ≤ 8,5"
	reOperatorValue = regexp.MustCompile(`(?i)([\p{L}][\p{L}\s\-]{0,40}?)\s*(≤|>=|=<|=>|≥|<=|<|>)\s*(\d+(?:[.,]\d+)?)(?:\s*([\p{L}/%°]+|°C|°F|K))?`)

	// property word-operator value [unit]: "сульфаты не более 300 мг/л", "pH не более 8,5"
	reWordOperator = regexp.MustCompile(`(?i)([\p{L}][\p{L}\s\-]{0,40}?)\s*(не более|не больше|до|меньше чем|меньше|не менее|не меньше|больше чем|больше)\s*(\d+(?:[.,]\d+)?)(?:\s*([\p{L}/%°]+|°C|°F|K))?`)

	// range: "от 100 до 300 мг/л"
	reRange = regexp.MustCompile(`(?i)от\s*(\d+(?:[.,]\d+)?)\s*до\s*(\d+(?:[.,]\d+)?)\s*([\p{L}/%°]+|°C|°F|K)`)
)

// extractNumericConstraints извлекает числовые ограничения из текста запроса.
func extractNumericConstraints(query string) []NumericConstraint {
	seen := make(map[string]struct{})
	var constraints []NumericConstraint

	add := func(c NumericConstraint) {
		c.Property = strings.ToLower(cleanProperty(c.Property))
		c.Unit = strings.ToLower(c.Unit)
		if c.Property == "" {
			return
		}
		key := fmt.Sprintf("%s|%s|%.4g|%.4g|%s", c.Property, c.Operator, c.Value, c.Value2, c.Unit)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		constraints = append(constraints, c)
	}

	for _, m := range reOperatorValue.FindAllStringSubmatch(query, -1) {
		prop := strings.TrimSpace(m[1])
		op := normalizeOperator(m[2])
		val := parseFloat(m[3])
		unit := strings.TrimSpace(m[4])
		if prop != "" && op != "" {
			add(NumericConstraint{Property: prop, Operator: op, Value: val, Unit: unit})
		}
	}

	for _, m := range reWordOperator.FindAllStringSubmatch(query, -1) {
		prop := strings.TrimSpace(m[1])
		op := wordToOperator(m[2])
		val := parseFloat(m[3])
		unit := strings.TrimSpace(m[4])
		if prop != "" && op != "" {
			add(NumericConstraint{Property: prop, Operator: op, Value: val, Unit: unit})
		}
	}

	for _, m := range reRange.FindAllStringSubmatch(query, -1) {
		val1 := parseFloat(m[1])
		val2 := parseFloat(m[2])
		unit := strings.TrimSpace(m[3])
		prop := inferProperty(query, m[0])
		add(NumericConstraint{Property: prop, Operator: "between", Value: val1, Value2: val2, Unit: unit})
	}

	return constraints
}

// enhanceQueryWithNumericConstraints добавляет в запрос инструкции для LLM,
// чтобы он учитывал найденные числовые ограничения.
func enhanceQueryWithNumericConstraints(query string) string {
	constraints := extractNumericConstraints(query)
	if len(constraints) == 0 {
		return query
	}

	var b strings.Builder
	b.WriteString(query)
	b.WriteString("\n\n")
	b.WriteString("При формировании ответа строго учитывай следующие числовые ограничения из вопроса. ")
	b.WriteString("Используй только источники, значения в которых соответствуют этим ограничениям. ")
	b.WriteString("Если источник противоречит ограничению — не используй его. ")
	b.WriteString("Для каждого упомянутого решения укажи конкретные числовые параметры, найденные в источниках, и единицы измерения.\n")
	for _, c := range constraints {
		b.WriteString("- ")
		b.WriteString(c.String())
		b.WriteString("\n")
	}

	return b.String()
}

func normalizeOperator(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "≤", "<=", "=<":
		return "<="
	case "<":
		return "<"
	case "≥", ">=", "=>":
		return ">="
	case ">":
		return ">"
	default:
		return ""
	}
}

func wordToOperator(op string) string {
	switch strings.ToLower(strings.TrimSpace(op)) {
	case "не более", "не больше", "до", "меньше чем", "меньше":
		return "<="
	case "не менее", "не меньше", "больше чем", "больше":
		return ">="
	default:
		return ""
	}
}

func parseFloat(s string) float64 {
	s = strings.ReplaceAll(s, ",", ".")
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return v
}

var prepositions = map[string]struct{}{
	"в": {}, "на": {}, "по": {}, "из": {}, "с": {}, "к": {}, "о": {},
	"об": {}, "под": {}, "над": {}, "при": {}, "для": {}, "от": {},
	"до": {}, "через": {}, "за": {}, "про": {}, "у": {}, "около": {},
	"между": {}, "перед": {}, "после": {}, "ввиду": {}, "вопреки": {},
}

func inferProperty(query, matched string) string {
	idx := strings.Index(strings.ToLower(query), strings.ToLower(matched))
	if idx <= 0 {
		return "значение"
	}

	before := strings.TrimSpace(query[:idx])
	words := strings.Fields(before)

	// Берём последние значащие слова (до 3), исключая предлоги.
	var parts []string
	for i := len(words) - 1; i >= 0 && len(parts) < 3; i-- {
		w := cleanProperty(words[i])
		if w == "" {
			continue
		}
		if _, isPrep := prepositions[w]; isPrep {
			break
		}
		parts = append([]string{w}, parts...)
	}

	if len(parts) > 0 {
		return strings.Join(parts, " ")
	}
	return "значение"
}

// buildNumericFilters объединяет явные фильтры из запроса и фильтры,
// извлечённые из текста вопроса, в единый список для поиска в БД.
func buildNumericFilters(req AskRequest) []models.NumericFactFilter {
	seen := make(map[string]struct{})
	var filters []models.NumericFactFilter

	add := func(f models.NumericFactFilter) {
		key := fmt.Sprintf("%s|%.4g|%.4g|%s", f.Property, f.Min, f.Max, f.Unit)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		filters = append(filters, f)
	}

	for _, f := range req.NumericFilters {
		add(models.NumericFactFilter{
			Property: strings.ToLower(strings.TrimSpace(f.Property)),
			Min:      f.Min,
			Max:      f.Max,
			Unit:     strings.ToLower(strings.TrimSpace(f.Unit)),
		})
	}

	for _, c := range extractNumericConstraints(req.Query) {
		var min, max float64
		switch c.Operator {
		case "<=", "<":
			max = c.Value
		case ">=", ">":
			min = c.Value
		case "between":
			min = c.Value
			max = c.Value2
		default:
			min = c.Value
			max = c.Value
		}
		add(models.NumericFactFilter{
			Property: c.Property,
			Min:      min,
			Max:      max,
			Unit:     c.Unit,
		})
	}

	return filters
}

// enhanceQueryWithDocumentFilter добавляет в запрос инструкцию использовать
// только документы из переданного списка.
func enhanceQueryWithDocumentFilter(query string, docIDs []string) string {
	var b strings.Builder
	b.WriteString(query)
	b.WriteString("\n\n")
	b.WriteString("При ответе используй информацию ТОЛЬКО из следующих документов (их идентификаторы): ")
	b.WriteString(strings.Join(docIDs, ", "))
	b.WriteString(". Если документ не из этого списка — не используй его.")
	return b.String()
}

// enrichResponseReferences заменяет UUID источников в тексте ответа LightRAG
// на реальные имена файлов, сохраняет номера ссылок [N] и удаляет
// встроенный markdown-блок References, чтобы не дублировать нижний список источников.
func enrichResponseReferences(response string, refs json.RawMessage) (string, json.RawMessage) {
	if len(refs) == 0 {
		return response, refs
	}

	var enriched []EnrichedReference
	if err := json.Unmarshal(refs, &enriched); err != nil {
		return response, refs
	}

	for i := range enriched {
		if enriched[i].Filename == "" {
			continue
		}
		response = strings.ReplaceAll(response, enriched[i].ID+".md", enriched[i].Filename)
		response = strings.ReplaceAll(response, enriched[i].ID, enriched[i].Filename)
	}

	assignReferenceNumbers(response, enriched)
	response = cleanReferencesBlock(response)

	updatedRefs, err := json.Marshal(enriched)
	if err != nil {
		return response, refs
	}
	return response, updatedRefs
}

// assignReferenceNumbers парсит markdown-блок References/Источники и заполняет Number
// в enriched references, чтобы номера в тексте ответа совпадали со списком источников.
func assignReferenceNumbers(response string, refs []EnrichedReference) {
	reBlock := regexp.MustCompile(`(?im)^#{1,3}\s*(References|Источники|Ссылки)\s*\n((?:^\s*[-*]\s.*\n?)+)`)
	m := reBlock.FindStringSubmatch(response)
	if m == nil {
		return
	}

	reLine := regexp.MustCompile(`(?i)^\s*[-*]\s*\[(\d+)\]\s*(.+?)\s*$`)
	numberByName := make(map[string]int)
	normalize := func(s string) string {
		s = strings.TrimSuffix(s, ".md")
		s = strings.TrimRight(s, ". ")
		return strings.TrimSpace(s)
	}
	for _, line := range strings.Split(m[2], "\n") {
		lm := reLine.FindStringSubmatch(line)
		if lm == nil {
			continue
		}
		num, _ := strconv.Atoi(lm[1])
		name := normalize(lm[2])
		numberByName[name] = num
	}

	for i := range refs {
		candidates := []string{
			normalize(refs[i].Filename),
			normalize(refs[i].SourcePath),
			refs[i].ID,
			refs[i].ID + ".md",
		}
		for _, cand := range candidates {
			if cand == "" {
				continue
			}
			if num, ok := numberByName[cand]; ok {
				refs[i].Number = num
				break
			}
			// Пробуем нормализованный вариант имени из блока.
			if num, ok := numberByName[normalize(cand)]; ok {
				refs[i].Number = num
				break
			}
		}
	}
}

// cleanReferencesBlock удаляет markdown-блок References/Источники/Ссылки из текста ответа.
func cleanReferencesBlock(response string) string {
	re := regexp.MustCompile(`(?im)^#{1,3}\s*(References|Источники|Ссылки)\s*\n(?:^\s*[-*]\s.*\n?)+`)
	return strings.TrimSpace(re.ReplaceAllString(response, ""))
}

func cleanProperty(s string) string {
	s = strings.TrimSpace(s)
	stopWords := map[string]bool{
		"если": true, "при": true, "где": true, "и": true, "или": true,
		"в": true, "с": true, "со": true, "по": true, "для": true, "от": true, "до": true,
		"какие": true, "какой": true, "каких": true, "какое": true, "какая": true,
		"методы": true, "метод": true, "способы": true, "способ": true,
		"варианты": true, "вариант": true, "решения": true, "решение": true,
		"применяются": true, "подходят": true, "используются": true,
		"сплавы": true, "сплав": true, "материалы": true, "материал": true,
	}

	words := strings.Fields(s)
	for len(words) > 0 && stopWords[strings.ToLower(words[0])] {
		words = words[1:]
	}
	if len(words) == 0 {
		return ""
	}
	return strings.Join(words, " ")
}
