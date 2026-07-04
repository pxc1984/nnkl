package data

import (
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/pxc1984/nnkl-backend/api"
)

// GraphNodeType represents the canonical node types used by the frontend legend.
type GraphNodeType string

const (
	GraphNodeTypeMaterial    GraphNodeType = "Material"
	GraphNodeTypeProcess     GraphNodeType = "Process"
	GraphNodeTypeEquipment   GraphNodeType = "Equipment"
	GraphNodeTypeProperty    GraphNodeType = "Property"
	GraphNodeTypeExperiment  GraphNodeType = "Experiment"
	GraphNodeTypePublication GraphNodeType = "Publication"
	GraphNodeTypeExpert      GraphNodeType = "Expert"
	GraphNodeTypeFacility    GraphNodeType = "Facility"
	GraphNodeTypeUnknown     GraphNodeType = "Unknown"
)

type GraphRequest struct {
	Query string `json:"query" binding:"required,min=3"`
	Mode  string `json:"mode"`
}

type GraphNode struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
}

type GraphEdge struct {
	Source      string  `json:"source"`
	Target      string  `json:"target"`
	Label       string  `json:"label"`
	Description string  `json:"description,omitempty"`
	Weight      float64 `json:"weight,omitempty"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Mode  string      `json:"mode"`
}

var (
	// typeMapping maps LightRAG / LLM entity types to our canonical legend types.
	typeMapping = map[string]GraphNodeType{
		"material":   GraphNodeTypeMaterial,
		"substance":  GraphNodeTypeMaterial,
		"chemical":   GraphNodeTypeMaterial,
		"compound":   GraphNodeTypeMaterial,
		"element":    GraphNodeTypeMaterial,
		"alloy":      GraphNodeTypeMaterial,
		"ore":        GraphNodeTypeMaterial,
		"mineral":    GraphNodeTypeMaterial,
		"metal":      GraphNodeTypeMaterial,
		"metall":     GraphNodeTypeMaterial,
		"металл":     GraphNodeTypeMaterial,
		"сплав":      GraphNodeTypeMaterial,
		"руда":       GraphNodeTypeMaterial,
		"минерал":    GraphNodeTypeMaterial,
		"вещество":   GraphNodeTypeMaterial,
		"химикат":    GraphNodeTypeMaterial,
		"соединение": GraphNodeTypeMaterial,
		"элемент":    GraphNodeTypeMaterial,

		"process":    GraphNodeTypeProcess,
		"method":     GraphNodeTypeProcess,
		"technique":  GraphNodeTypeProcess,
		"operation":  GraphNodeTypeProcess,
		"technology": GraphNodeTypeProcess,
		"procedure":  GraphNodeTypeProcess,
		"approach":   GraphNodeTypeProcess,
		"workflow":   GraphNodeTypeProcess,
		"процесс":    GraphNodeTypeProcess,
		"метод":      GraphNodeTypeProcess,
		"технология": GraphNodeTypeProcess,
		"способ":     GraphNodeTypeProcess,
		"операция":   GraphNodeTypeProcess,
		"подход":     GraphNodeTypeProcess,

		"equipment":    GraphNodeTypeEquipment,
		"device":       GraphNodeTypeEquipment,
		"machine":      GraphNodeTypeEquipment,
		"apparatus":    GraphNodeTypeEquipment,
		"tool":         GraphNodeTypeEquipment,
		"instrument":   GraphNodeTypeEquipment,
		"gear":         GraphNodeTypeEquipment,
		"installation": GraphNodeTypeEquipment,
		"оборудование": GraphNodeTypeEquipment,
		"устройство":   GraphNodeTypeEquipment,
		"машина":       GraphNodeTypeEquipment,
		"аппарат":      GraphNodeTypeEquipment,
		"инструмент":   GraphNodeTypeEquipment,
		"печь":         GraphNodeTypeEquipment,
		"ванна":        GraphNodeTypeEquipment,
		"реактор":      GraphNodeTypeEquipment,
		"электрод":     GraphNodeTypeEquipment,
		"горелка":      GraphNodeTypeEquipment,
		"конвертер":    GraphNodeTypeEquipment,
		"фильтр":       GraphNodeTypeEquipment,
		"насос":        GraphNodeTypeEquipment,

		"property":       GraphNodeTypeProperty,
		"parameter":      GraphNodeTypeProperty,
		"characteristic": GraphNodeTypeProperty,
		"metric":         GraphNodeTypeProperty,
		"attribute":      GraphNodeTypeProperty,
		"feature":        GraphNodeTypeProperty,
		"indicator":      GraphNodeTypeProperty,
		"value":          GraphNodeTypeProperty,
		"свойство":       GraphNodeTypeProperty,
		"параметр":       GraphNodeTypeProperty,
		"характеристика": GraphNodeTypeProperty,
		"показатель":     GraphNodeTypeProperty,
		"значение":       GraphNodeTypeProperty,
		"category":       GraphNodeTypeProperty,
		"concept":        GraphNodeTypeProperty,
		"категория":      GraphNodeTypeProperty,
		"понятие":        GraphNodeTypeProperty,

		"experiment":    GraphNodeTypeExperiment,
		"study":         GraphNodeTypeExperiment,
		"trial":         GraphNodeTypeExperiment,
		"test":          GraphNodeTypeExperiment,
		"research":      GraphNodeTypeExperiment,
		"investigation": GraphNodeTypeExperiment,
		"assay":         GraphNodeTypeExperiment,
		"эксперимент":   GraphNodeTypeExperiment,
		"исследование":  GraphNodeTypeExperiment,
		"испытание":     GraphNodeTypeExperiment,
		"тест":          GraphNodeTypeExperiment,
		"опыт":          GraphNodeTypeExperiment,
		"event":         GraphNodeTypeExperiment,
		"событие":       GraphNodeTypeExperiment,

		"publication": GraphNodeTypePublication,
		"paper":       GraphNodeTypePublication,
		"article":     GraphNodeTypePublication,
		"report":      GraphNodeTypePublication,
		"patent":      GraphNodeTypePublication,
		"document":    GraphNodeTypePublication,
		"thesis":      GraphNodeTypePublication,
		"review":      GraphNodeTypePublication,
		"публикация":  GraphNodeTypePublication,
		"статья":      GraphNodeTypePublication,
		"отчёт":       GraphNodeTypePublication,
		"патент":      GraphNodeTypePublication,
		"документ":    GraphNodeTypePublication,
		"диссертация": GraphNodeTypePublication,
		"обзор":       GraphNodeTypePublication,

		"person":        GraphNodeTypeExpert,
		"people":        GraphNodeTypeExpert,
		"author":        GraphNodeTypeExpert,
		"researcher":    GraphNodeTypeExpert,
		"scientist":     GraphNodeTypeExpert,
		"engineer":      GraphNodeTypeExpert,
		"expert":        GraphNodeTypeExpert,
		"specialist":    GraphNodeTypeExpert,
		"employee":      GraphNodeTypeExpert,
		"organization":  GraphNodeTypeExpert,
		"company":       GraphNodeTypeExpert,
		"institution":   GraphNodeTypeExpert,
		"institute":     GraphNodeTypeExpert,
		"laboratory":    GraphNodeTypeExpert,
		"lab":           GraphNodeTypeExpert,
		"center":        GraphNodeTypeExpert,
		"university":    GraphNodeTypeExpert,
		"человек":       GraphNodeTypeExpert,
		"личность":      GraphNodeTypeExpert,
		"автор":         GraphNodeTypeExpert,
		"исследователь": GraphNodeTypeExpert,
		"учёный":        GraphNodeTypeExpert,
		"инженер":       GraphNodeTypeExpert,
		"эксперт":       GraphNodeTypeExpert,
		"специалист":    GraphNodeTypeExpert,
		"организация":   GraphNodeTypeExpert,
		"компания":      GraphNodeTypeExpert,
		"институт":      GraphNodeTypeExpert,
		"лаборатория":   GraphNodeTypeExpert,
		"центр":         GraphNodeTypeExpert,
		"университет":   GraphNodeTypeExpert,

		"facility":      GraphNodeTypeFacility,
		"plant":         GraphNodeTypeFacility,
		"factory":       GraphNodeTypeFacility,
		"mine":          GraphNodeTypeFacility,
		"smelter":       GraphNodeTypeFacility,
		"refinery":      GraphNodeTypeFacility,
		"mill":          GraphNodeTypeFacility,
		"site":          GraphNodeTypeFacility,
		"location":      GraphNodeTypeFacility,
		"place":         GraphNodeTypeFacility,
		"geo":           GraphNodeTypeFacility,
		"gpe":           GraphNodeTypeFacility,
		"country":       GraphNodeTypeFacility,
		"city":          GraphNodeTypeFacility,
		"region":        GraphNodeTypeFacility,
		"объект":        GraphNodeTypeFacility,
		"площадка":      GraphNodeTypeFacility,
		"завод":         GraphNodeTypeFacility,
		"фабрика":       GraphNodeTypeFacility,
		"шахта":         GraphNodeTypeFacility,
		"рудник":        GraphNodeTypeFacility,
		"комбинат":      GraphNodeTypeFacility,
		"гок":           GraphNodeTypeFacility,
		"месторождение": GraphNodeTypeFacility,
		"страна":        GraphNodeTypeFacility,
		"город":         GraphNodeTypeFacility,
		"регион":        GraphNodeTypeFacility,
		"гео":           GraphNodeTypeFacility,
	}

	// nameSuffixRules map Russian/English suffixes to canonical types.
	nameSuffixRules = []struct {
		suffixes []string
		typ      GraphNodeType
	}{
		{[]string{"печь", "ванна", "реактор", "электрод", "горелка", "конвертер", "фильтр", "насос", "сите", "мельница", "дробилка", "машина", "аппарат"}, GraphNodeTypeEquipment},
		{[]string{"завод", "фабрика", "шахта", "рудник", "комбинат", "гок", "месторождение", "площадка", "страна", "город", "регион"}, GraphNodeTypeFacility},
		{[]string{"процесс", "метод", "технология", "способ", "операция", "подход"}, GraphNodeTypeProcess},
		{[]string{"сплав", "металл", "руда", "минерал", "сульфат", "хлорид", "оксид", "карбонат", "кислота", "ион", "вещество", "соединение", "элемент", "концентрат", "катод", "анод", "шлак", "штейн", "флюс"}, GraphNodeTypeMaterial},
		{[]string{"исследование", "эксперимент", "испытание", "тест", "опыт"}, GraphNodeTypeExperiment},
		{[]string{"статья", "отчёт", "патент", "диссертация", "обзор", "публикация", "документ", "publication", "paper", "article", "report", "patent", "thesis", "review"}, GraphNodeTypePublication},
	}

	nameContainsRules = []struct {
		fragments []string
		typ       GraphNodeType
	}{
		{[]string{"печь", "ванна", "реактор", "электрод", "горелка", "конвертер", "фильтр", "насос", "сите", "мельница", "дробилка", "машина", "аппарат", "оборудование", "устройство"}, GraphNodeTypeEquipment},
		{[]string{"завод", "фабрика", "шахта", "рудник", "комбинат", "гок", "месторождение", "площадка", "страна", "город", "регион"}, GraphNodeTypeFacility},
		{[]string{"процесс", "метод", "технология", "способ", "операция", "подход"}, GraphNodeTypeProcess},
		{[]string{"сплав", "металл", "руда", "минерал", "сульфат", "хлорид", "оксид", "карбонат", "кислота", "ион", "вещество", "соединение", "элемент", "химикат", "концентрат", "катод", "анод", "шлак", "штейн", "флюс"}, GraphNodeTypeMaterial},
		{[]string{"исследование", "эксперимент", "испытание", "тест", "опыт", "событие", "study", "research", "experiment", "test", "trial", "assay"}, GraphNodeTypeExperiment},
		{[]string{"статья", "отчёт", "патент", "диссертация", "обзор", "публикация", "документ"}, GraphNodeTypePublication},
		{[]string{"компания", "лаборатория", "организация", "институт", "центр", "университет"}, GraphNodeTypeExpert},
	}

	// personNamePattern matches names like "Иванов И.И.", "R.T. Jones", "Евграфова А.К.".
	personNamePattern = regexp.MustCompile(`^([А-ЯA-Z][а-яa-z]+\s+[А-ЯA-Z][\.\s]?[А-ЯA-Z]?[\.]?\s*)$|^([А-ЯA-Z][а-яa-z]+\s+[А-ЯA-Z][а-яa-z]+)$`)
)

func normalizeType(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func looksLikePersonName(name string) bool {
	trimmed := strings.TrimSpace(name)
	if personNamePattern.MatchString(trimmed) {
		return true
	}
	// Heuristic: two or three capitalized words, no digits.
	words := strings.Fields(trimmed)
	if len(words) < 1 || len(words) > 4 {
		return false
	}
	for _, w := range words {
		runes := []rune(w)
		if len(runes) == 0 {
			return false
		}
		if !unicode.IsUpper(runes[0]) {
			return false
		}
		for _, r := range runes[1:] {
			if unicode.IsDigit(r) {
				return false
			}
		}
	}
	return len(words) >= 2
}

func mapEntityType(rawType, name string) GraphNodeType {
	key := normalizeType(rawType)
	if key == "" {
		return inferTypeFromName(name)
	}
	if mapped, ok := typeMapping[key]; ok {
		return mapped
	}
	// Try base form without trailing 's' for plurals.
	if strings.HasSuffix(key, "s") {
		if mapped, ok := typeMapping[key[:len(key)-1]]; ok {
			return mapped
		}
	}
	return inferTypeFromName(name)
}

func inferTypeFromName(name string) GraphNodeType {
	lower := strings.ToLower(strings.TrimSpace(name))
	if lower == "" {
		return GraphNodeTypeUnknown
	}

	// Suffix rules first.
	for _, rule := range nameSuffixRules {
		for _, suffix := range rule.suffixes {
			if strings.HasSuffix(lower, suffix) {
				return rule.typ
			}
		}
	}

	// Contains rules.
	for _, rule := range nameContainsRules {
		for _, fragment := range rule.fragments {
			if strings.Contains(lower, fragment) {
				return rule.typ
			}
		}
	}

	if looksLikePersonName(name) {
		return GraphNodeTypeExpert
	}

	return GraphNodeTypeUnknown
}

func convertLightRAGDataToGraph(resp *LightRAGQueryDataResponse, mode string) GraphResponse {
	nodes := make([]GraphNode, 0, len(resp.Data.Entities))
	nodeIDs := make(map[string]struct{})
	for _, entity := range resp.Data.Entities {
		id := strings.TrimSpace(entity.EntityName)
		if id == "" {
			continue
		}
		if _, exists := nodeIDs[id]; exists {
			continue
		}
		nodeIDs[id] = struct{}{}
		nodes = append(nodes, GraphNode{
			ID:          id,
			Label:       id,
			Type:        string(mapEntityType(entity.EntityType, id)),
			Description: strings.TrimSpace(entity.Description),
		})
	}

	edges := make([]GraphEdge, 0, len(resp.Data.Relationships))
	for _, relation := range resp.Data.Relationships {
		src := strings.TrimSpace(relation.SrcID)
		tgt := strings.TrimSpace(relation.TgtID)
		if src == "" || tgt == "" {
			continue
		}
		if _, exists := nodeIDs[src]; !exists {
			continue
		}
		if _, exists := nodeIDs[tgt]; !exists {
			continue
		}
		label := strings.TrimSpace(relation.Keywords)
		if label == "" {
			label = "связан_с"
		}
		edges = append(edges, GraphEdge{
			Source:      src,
			Target:      tgt,
			Label:       label,
			Description: strings.TrimSpace(relation.Description),
			Weight:      relation.Weight,
		})
	}

	return GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Mode:  mode,
	}
}

func (a *DataAPI) graph(c *gin.Context) {
	var req GraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.RespondError(c, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	if !a.lightrag.IsConfigured() {
		api.RespondError(c, http.StatusServiceUnavailable, "lightrag service is not configured", "service_unavailable")
		return
	}

	mode := req.Mode
	if mode == "" {
		mode = "hybrid"
	}

	resp, err := a.lightrag.QueryData(c.Request.Context(), req.Query, mode)
	if err != nil {
		api.RespondError(c, http.StatusServiceUnavailable, "failed to query knowledge graph: "+err.Error(), "service_unavailable")
		return
	}

	c.JSON(http.StatusOK, convertLightRAGDataToGraph(resp, mode))
}
