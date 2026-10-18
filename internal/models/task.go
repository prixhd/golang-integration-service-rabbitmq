package models

const (
	StatusInProgress = "in_progress"
	StatusReady      = "ready"
)

type Task struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Language string `json:"language"`
	Status   string `json:"status"`
	Result   string `json:"result"`
}
