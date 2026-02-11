package status

import (
	"cgl/obj"
	"encoding/json"
	"fmt"
)

// ParseGameResponse parses raw AI JSON text into the response message fields.
// It unmarshals the AI response, converts the flat status map back to ordered
// []StatusField, and populates response.Message, StatusFields, and ImagePrompt.
// actionStatusFields provides fallback values if the AI omits a field.
func ParseGameResponse(responseText string, sessionStatusFields string, actionStatusFields []obj.StatusField, response *obj.GameSessionMessage) error {
	var aiResp obj.GameSessionMessageAi
	if err := json.Unmarshal([]byte(responseText), &aiResp); err != nil {
		return fmt.Errorf("failed to parse game response: %w", err)
	}

	fieldNames := FieldNames(sessionStatusFields)
	response.Message = aiResp.Message
	response.StatusFields = MapToFields(aiResp.Status, fieldNames, FieldsToMap(actionStatusFields))
	response.ImagePrompt = aiResp.ImagePrompt

	return nil
}
