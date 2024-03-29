package obj

type User struct {
	ID                uint   `json:"id"`
	Name              string `json:"name"`
	OpenAiKeyPublish  string `json:"openaiKeyPublish"`
	OpenAiKeyPersonal string `json:"openaiKeyPersonal"`
}

type Game struct {
	ID                  uint   `json:"id"`
	Title               string `json:"title"`
	TitleImage          []byte
	Description         string        `json:"description"`
	Scenario            string        `json:"scenario"`
	SessionStartSyscall string        `json:"sessionStartSyscall"`
	StatusFields        []StatusField `json:"statusFields"`
	ImageStyle          string        `json:"imageStyle"`
	SharePlayActive     bool          `json:"sharePlayActive"`
	SharePlayHash       string        `json:"sharePlayHash"`
	ShareEditActive     bool          `json:"shareEditActive"`
	ShareEditHash       string        `json:"shareEditHash"`
	UserId              uint          `json:"userId"`
	UserName            string        `json:"userName"`
}

type Session struct {
	ID                    uint   `json:"id"`
	GameID                uint   `json:"gameId"`
	UserID                uint   `json:"userId"`
	AssistantID           string `json:"assistantId"`
	AssistantInstructions string `json:"assistantInstructions"`
	ThreadID              string `json:"threadId"`
	Hash                  string `json:"hash"`
}

type Chapter struct {
	SessionID   uint   `json:"sessionId"`
	Chapter     uint   `json:"chapter"`
	Input       string `json:"input"`
	Output      string `json:"output"`
	ImagePrompt string `json:"imagePrompt"`
	Image       []byte `json:"image"`
}

type StatusField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

const GameInputTypeAction = "player-action"
const GameInputTypeIntro = "intro"
const GameOutputTypeError = "error"
const GameOutputTypeStory = "story"

type GameActionInput struct {
	ChapterId uint          `json:"-"`
	Type      string        `json:"type"`
	Message   string        `json:"action"`
	Status    []StatusField `json:"status"`
}

/*
{
  "story": "You opened the red door with the key. The key stuck in the door. You're now outside the castle.",
 "status": {{STATUS}},
"image":"a castle in the background, green grass, late afternoon"
}
*/

type GameActionOutput struct {
	ChapterId             uint          `json:"chapterId"`
	SessionHash           string        `json:"sessionHash"`
	Type                  string        `json:"type"`
	Story                 string        `json:"story"`
	Status                []StatusField `json:"status"`
	Image                 string        `json:"image"`
	Error                 string        `json:"error"`
	RawInput              string        `json:"rawInput"`
	RawOutput             string        `json:"rawOutput"`
	AssistantInstructions string        `json:"assistantInstructions"`
}
