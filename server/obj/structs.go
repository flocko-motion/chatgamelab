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
	Description         string `json:"description"`
	Scenario            string `json:"scenario"`
	SessionStartSyscall string `json:"sessionStartSyscall"`
	PostActionSyscall   string `json:"postActionSyscall"`
	ImageStyle          string `json:"imageStyle"`
	SharePlayActive     bool   `json:"sharePlayActive"`
	SharePlayHash       string `json:"sharePlayHash"`
	ShareEditActive     bool   `json:"shareEditActive"`
	ShareEditHash       string `json:"shareEditHash"`
	UserId              uint   `json:"userId"`
	UserName            string `json:"userName"`
}

type Session struct {
	ID          uint   `json:"id"`
	GameID      uint   `json:"gameId"`
	UserID      uint   `json:"userId"`
	AssistantID string `json:"assistantId"`
	ThreadID    string `json:"threadId"`
	Hash        string `json:"hash"`
}

type StatusField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
