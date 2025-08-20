package model

type BatchPayRequest struct {
	BatchPayID string  `json:"batchPayId"` // 批次 ID
	UIDs       []int64 `json:"uids"`       // 要处理的 uid 列表
}
