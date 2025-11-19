package model

import "strings"

// InternalModel is minimal payload for IG integration.
type InternalModel struct {
	ID           int      `json:"id"`
	ModelID      string   `json:"model_id"`
	DisplayName  string   `json:"display_name"`
	EnableGroups []string `json:"enable_groups"`
	ThinkMode    bool     `json:"think_mode"`
}

// GetModelsForInternal returns enabled models and associated groups.
func GetModelsForInternal() ([]InternalModel, error) {
	var models []struct {
		ID        int
		ModelName string
		Tags      string
	}
	if err := DB.Table("models").Select("id, model_name, tags").Where("status = ? AND deleted_at IS NULL", 1).Scan(&models).Error; err != nil {
		return nil, err
	}
	var abilityRows []struct {
		Model string
		Group string
	}
	if err := DB.Table("abilities").Select("model, `group`").Where("enabled = ?", true).Scan(&abilityRows).Error; err != nil {
		return nil, err
	}
	groupMap := make(map[string][]string)
	seen := make(map[string]map[string]struct{})
	for _, row := range abilityRows {
		if row.Model == "" || row.Group == "" {
			continue
		}
		if _, ok := seen[row.Model]; !ok {
			seen[row.Model] = make(map[string]struct{})
		}
		lower := strings.TrimSpace(row.Group)
		if _, exists := seen[row.Model][lower]; exists {
			continue
		}
		seen[row.Model][lower] = struct{}{}
		groupMap[row.Model] = append(groupMap[row.Model], lower)
	}
	result := make([]InternalModel, 0, len(models))
	for _, m := range models {
		entry := InternalModel{
			ID:           m.ID,
			ModelID:      m.ModelName,
			DisplayName:  m.ModelName,
			EnableGroups: groupMap[m.ModelName],
			ThinkMode:    hasThinkModeTag(m.Tags),
		}
		result = append(result, entry)
	}
	return result, nil
}

func hasThinkModeTag(tags string) bool {
	if strings.TrimSpace(tags) == "" {
		return false
	}
	parts := strings.Split(tags, ",")
	for _, part := range parts {
		if strings.EqualFold(strings.TrimSpace(part), "thinkmode") {
			return true
		}
	}
	return false
}
