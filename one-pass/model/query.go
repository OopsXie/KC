package model

// QueryUserAmountRequest 查询用户余额请求
type QueryUserAmountRequest []int64

// QueryUserAmountResponse 查询用户余额响应
type QueryUserAmountResponse struct {
	Code      int              `json:"code"`
	Msg       string           `json:"msg"`
	RequestID string           `json:"requestId"`
	Data      []UserAmountData `json:"data"`
}

// UserAmountData 用户余额数据
type UserAmountData struct {
	UID    int64   `json:"uid"`
	Amount float64 `json:"amount,omitempty"`
	Msg    string  `json:"msg,omitempty"`
}
