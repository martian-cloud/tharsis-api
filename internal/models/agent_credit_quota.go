package models

import "time"

// AgentCreditQuota tracks monthly LLM credit usage for a user.
type AgentCreditQuota struct {
	Metadata     ResourceMetadata
	UserID       string
	MonthDate    time.Time // 1st of the month
	TotalCredits float64
}
