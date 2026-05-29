package models

import "time"

type TempAccess struct {
	TargetID  string    `bson:"target_id" json:"target_id"`
	GuildID   string    `bson:"guild_id" json:"guild_id"`
	Type      string    `bson:"type" json:"type"`
	ExpiresAt time.Time `bson:"expires_at" json:"expires_at"`
}

type TaskProgress struct {
	TaskID        string    `bson:"task_id" json:"task_id"`
	CurrentValue  int       `bson:"current_value" json:"current_value"`
	Completed     bool      `bson:"completed" json:"completed"`
	LastCompleted time.Time `bson:"last_completed,omitempty" json:"last_completed,omitempty"`
}

type UserProfile struct {
	UserID            string         `bson:"user_id" json:"user_id"`
	MessageCount      int            `bson:"message_count" json:"message_count"`
	ShareCount        int            `bson:"share_count" json:"share_count"`
	XP                int            `bson:"xp" json:"xp"`
	Badges            []string       `bson:"badges" json:"badges"`
	Tasks             []TaskProgress `bson:"tasks" json:"tasks"`
	TempAccesses      []TempAccess   `bson:"temp_accesses" json:"temp_accesses"`
	DailyMessageCount int            `bson:"daily_message_count" json:"daily_message_count"`
	DailyShareCount   int            `bson:"daily_share_count" json:"daily_share_count"`
	LastResetAt       time.Time      `bson:"last_reset_at" json:"last_reset_at"`
	LastMessageAt     time.Time      `bson:"last_message_at" json:"last_message_at"`
	UpdatedAt         time.Time      `bson:"updated_at" json:"updated_at"`
}
