package status

import (
	"cgl/obj"
	"encoding/json"
)

// ParseGameResponse parses raw AI JSON text into the response message fields.
// It unmarshals the AI response, converts the flat status map back to ordered
// []StatusField, and populates response.Plot, StatusFields, and ImagePrompt.
// The plot outline is stored in Plot (not Message) — Message is only set later by ExpandStory with the full prose.
// actionStatusFields provides fallback values if the AI omits a field.
func ParseGameResponse(responseText string, sessionStatusFields string, actionStatusFields []obj.StatusField, response *obj.GameSessionMessage) error {
	var aiResp obj.GameSessionMessageAi
	if err := json.Unmarshal([]byte(responseText), &aiResp); err != nil {
		return obj.WrapError(obj.ErrCodeAiError, "failed to parse game response", err)
	}

	fieldNames := FieldNames(sessionStatusFields)
	plot := aiResp.Message
	response.Plot = &plot
	response.StatusFields = MapToFields(aiResp.Status, fieldNames, FieldsToMap(actionStatusFields))
	response.ImagePrompt = aiResp.ImagePrompt

	return nil
}
