package rabbitmq

type EmailMessage struct {
	To    string `json:"to"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type VideoMessage struct {
	VideoID    string   `json:"video_id"`
	UserID     string   `json:"user_id"`
	InputPath  string   `json:"input_path"`
	OutputPath string   `json:"output_path"`
	Formats    []string `json:"formats"`
}

type SaveVideoMessage struct {
	Key      string `json:"key"`
	UserID   string `json:"user_id"`
	Filename string `json:"filename"`
}
