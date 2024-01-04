package obj

type User struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
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
	User                *User  `json:"user"`
}
