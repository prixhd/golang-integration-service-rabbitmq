package models

type TaskMessage struct {
	TaskID     string `json:"task_id"`
	Translator string `json:"translator"`
	Code       string `json:"code"`
}
