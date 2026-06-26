package queue

type EmailMessage struct {
	To    string `json:"to"`
	Name  string `json:"name"`
	Token string `json:"token"`
}

type VideoMessage struct {
	UserID  string  `json:"user_id"`
	JobId   string  `json:"job_id"`
	Key     string  `json:"upload_key"`
	Start   float32 `json:"start_time"`
	End     float32 `json:"end_time"`
	Width   int     `json:"width"`
	FPS     int     `json:"fps"`
	Loop    bool    `json:"loop"`
	Retries int     `json:"retries"`
}

type SaveVideoMessage struct {
	Key      string `json:"key"`
	UserID   string `json:"user_id"`
	Filename string `json:"filename"`
	Retries  int    `json:"retries"`
}
