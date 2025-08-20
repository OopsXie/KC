package model

type UserBalance struct {
	UID    int64   `gorm:"primaryKey;index:idx_uid" json:"uid"`       // 添加索引
	Amount float64 `gorm:"not null;type:decimal(15,2)" json:"amount"` // 使用decimal类型避免浮点精度问题
}
