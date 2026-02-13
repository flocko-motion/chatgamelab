package game

import (
	"context"
	"encoding/json"
	"fmt"

	"cgl/game/ai"
	"cgl/lang"
	"cgl/log"
	"cgl/obj"
)

// TranslateGame translates a game's text content to the target language using AI.
// Returns the translated game, a mapping of original→translated status field names, and token usage.
func TranslateGame(ctx context.Context, session *obj.GameSession, game *obj.Game, targetLang string) (*obj.Game, map[string]string, obj.TokenUsage, error) {
	if session == nil || session.ApiKey == nil {
		return nil, nil, obj.TokenUsage{}, obj.ErrInvalidApiKey("session or API key is nil")
	}

	// Validate target language
	if !lang.IsValidLanguageCode(targetLang) {
		return nil, nil, obj.TokenUsage{}, obj.ErrValidationf("unsupported language code: %s", targetLang)
	}

	// Get language name for better AI context
	langName := lang.GetLanguageName(targetLang)
	log.Debug("translating game", "game_id", game.ID, "target_lang", targetLang, "lang_name", langName)

	// Get AI platform
	platform, err := ai.GetAiPlatform(session.AiPlatform)
	if err != nil {
		return nil, nil, obj.TokenUsage{}, obj.WrapError(obj.ErrCodeAiError, "failed to get AI platform", err)
	}

	// Prepare game content for translation
	// We translate: Name, Description, SystemMessageScenario, SystemMessageGameStart
	gameContent := map[string]string{
		"name":                   game.Name,
		"description":            game.Description,
		"systemMessageScenario":  game.SystemMessageScenario,
		"systemMessageGameStart": game.SystemMessageGameStart,
	}

	// Also translate status field names (default to empty array if missing)
	var statusFields []obj.StatusField
	if game.StatusFields == "" {
		game.StatusFields = "[]"
	}
	if err := json.Unmarshal([]byte(game.StatusFields), &statusFields); err == nil {
		for i, field := range statusFields {
			gameContent[fmt.Sprintf("statusField_%d_name", i)] = field.Name
		}
	}

	// Convert to JSON for AI
	contentJSON, err := json.Marshal(gameContent)
	if err != nil {
		return nil, nil, obj.TokenUsage{}, obj.WrapError(obj.ErrCodeServerError, "failed to marshal game content", err)
	}

	// Call AI translation
	log.Debug("calling AI to translate game content", "target_lang", targetLang)
	translatedJSON, usage, err := platform.Translate(ctx, session.ApiKey.Key, []string{string(contentJSON)}, targetLang)
	if err != nil {
		return nil, nil, usage, obj.WrapError(obj.ErrCodeAiError, "AI translation failed", err)
	}

	// Parse translated content
	var translatedContent map[string]string
	if err := json.Unmarshal([]byte(translatedJSON), &translatedContent); err != nil {
		return nil, nil, usage, obj.WrapError(obj.ErrCodeAiError, "failed to parse translated content", err)
	}

	// Create a copy of the game with translated content
	translatedGame := *game
	translatedGame.Name = getTranslatedField(translatedContent, "name", game.Name)
	translatedGame.Description = getTranslatedField(translatedContent, "description", game.Description)
	translatedGame.SystemMessageScenario = getTranslatedField(translatedContent, "systemMessageScenario", game.SystemMessageScenario)
	translatedGame.SystemMessageGameStart = getTranslatedField(translatedContent, "systemMessageGameStart", game.SystemMessageGameStart)

	// Translate status field names and build original→translated mapping
	fieldNameMap := make(map[string]string)
	if len(statusFields) > 0 {
		for i := range statusFields {
			key := fmt.Sprintf("statusField_%d_name", i)
			originalName := statusFields[i].Name
			translatedName := getTranslatedField(translatedContent, key, originalName)
			statusFields[i].Name = translatedName
			fieldNameMap[originalName] = translatedName
		}
		translatedStatusJSON, err := json.Marshal(statusFields)
		if err == nil {
			translatedGame.StatusFields = string(translatedStatusJSON)
		}
	}

	log.Debug("game translation completed", "game_id", game.ID, "target_lang", targetLang)
	return &translatedGame, fieldNameMap, usage, nil
}

// getTranslatedField retrieves a translated field from the map, falling back to original if not found
func getTranslatedField(translated map[string]string, key, fallback string) string {
	if val, ok := translated[key]; ok && val != "" {
		return val
	}
	return fallback
}
