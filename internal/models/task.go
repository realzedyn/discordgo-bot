package models

type Reward struct {
	Type     string `json:"type"`
	Value    string `json:"value"`
	Duration string `json:"duration"`
}

type TaskRequirement struct {
	Type         string `json:"type"`
	TargetValue  int    `json:"target_value"`
	ChannelID    string `json:"channel_id,omitempty"`
	RequiredRole string `json:"required_role,omitempty"`
}

type Task struct {
	ID           string          `json:"id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Requirements TaskRequirement `json:"requirements"`
	Rewards      []Reward        `json:"rewards"`
	IsRepeatable bool            `json:"is_repeatable"`
	Category     string          `json:"category"`
}
