package model

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/shopspring/decimal"
)

// UserTradeRequest 用户交易请求
type UserTradeRequest struct {
	SourceUID int64   `json:"sourceUid"` // 交易源账户（扣减账户）
	TargetUID int64   `json:"targetUid"` // 交易目标账户（添加账户）
	Amount    float64 `json:"amount"`    // 交易金额
}

// UserTradeRequestString 用字符串接收金额的交易请求
type UserTradeRequestString struct {
	SourceUID int64           `json:"sourceUid"` // 交易源账户（扣减账户）
	TargetUID int64           `json:"targetUid"` // 交易目标账户（添加账户）
	Amount    json.RawMessage `json:"amount"`    // 交易金额（支持字符串和数字格式）
}

// GetAmountBigFloat 获取 big.Float 格式的金额
func (r *UserTradeRequestString) GetAmountBigFloat() (*big.Float, error) {
	bigAmount := new(big.Float)
	bigAmount.SetPrec(128) // 设置高精度

	// Convert json.RawMessage to string, removing quotes if present
	amountStr := string(r.Amount)
	if len(amountStr) >= 2 && amountStr[0] == '"' && amountStr[len(amountStr)-1] == '"' {
		amountStr = amountStr[1 : len(amountStr)-1]
	}

	_, ok := bigAmount.SetString(amountStr)
	if !ok {
		return nil, fmt.Errorf("无效的金额格式: %s", amountStr)
	}
	return bigAmount, nil
}

// ValidateAmountPrecision 验证金额精度 - 使用字符串精确验证
func (r *UserTradeRequestString) ValidateAmountPrecision() error {
	// 获取原始金额字符串
	amountStr := string(r.Amount)

	// 去掉可能的引号
	if len(amountStr) >= 2 && amountStr[0] == '"' && amountStr[len(amountStr)-1] == '"' {
		amountStr = amountStr[1 : len(amountStr)-1]
	}

	// 使用 shopspring/decimal 进行精确解析
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return fmt.Errorf("无效的金额格式: %s", amountStr)
	}

	// 检查金额范围
	minAmount := decimal.NewFromFloat(0.01)
	maxAmount := decimal.NewFromFloat(10000)

	if amount.LessThan(minAmount) {
		return fmt.Errorf("交易金额不能小于0.01")
	}

	if amount.GreaterThan(maxAmount) {
		return fmt.Errorf("交易金额不能大于10000")
	}

	// 检查是否是 0.01 的整数倍 - 使用更精确的方法
	// 将金额乘以100，检查结果是否为整数
	cents := amount.Mul(decimal.NewFromInt(100))

	// 检查是否为整数 - 通过检查小数部分是否为0
	if !cents.Equal(cents.Truncate(0)) {
		return fmt.Errorf("交易金额最小单位为0.01，当前金额精度过高")
	}

	return nil
}

// GetAmountFloat64 获取 float64 格式的金额（用于向后兼容）
func (r *UserTradeRequestString) GetAmountFloat64() (float64, error) {
	bigAmount, err := r.GetAmountBigFloat()
	if err != nil {
		return 0, err
	}

	// 转换为精确的两位小数
	cents := new(big.Float).Mul(bigAmount, big.NewFloat(100))
	centsInt, _ := cents.Int64()
	return float64(centsInt) / 100.0, nil
}

// UserTradeRequestRaw 用于精确解析的交易请求
type UserTradeRequestRaw struct {
	SourceUID int64           `json:"sourceUid"` // 交易源账户（扣减账户）
	TargetUID int64           `json:"targetUid"` // 交易目标账户（添加账户）
	Amount    json.RawMessage `json:"amount"`    // 原始JSON金额字符串
}

// GetAmountFloat 获取float64格式的金额
func (r *UserTradeRequestRaw) GetAmountFloat() (float64, error) {
	var amount float64
	err := json.Unmarshal(r.Amount, &amount)
	return amount, err
}

// ValidateAmountPrecision 使用decimal进行精确验证金额精度
func (r *UserTradeRequestRaw) ValidateAmountPrecision() error {
	// 将原始JSON字符串转换为decimal.Decimal
	amountStr := string(r.Amount)

	// 去掉可能的引号
	if len(amountStr) >= 2 && amountStr[0] == '"' && amountStr[len(amountStr)-1] == '"' {
		amountStr = amountStr[1 : len(amountStr)-1]
	}

	// 使用 shopspring/decimal 进行精确解析
	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return fmt.Errorf("无效的金额格式: %s", amountStr)
	}

	// 检查金额范围
	minAmount := decimal.NewFromFloat(0.01)
	maxAmount := decimal.NewFromFloat(10000)

	if amount.LessThan(minAmount) {
		return fmt.Errorf("交易金额不能小于0.01")
	}

	if amount.GreaterThan(maxAmount) {
		return fmt.Errorf("交易金额不能大于10000")
	}

	// 检查是否是 0.01 的整数倍 - 使用更精确的方法
	// 将金额乘以100，检查结果是否为整数
	cents := amount.Mul(decimal.NewFromInt(100))

	// 检查是否为整数 - 通过检查小数部分是否为0
	if !cents.Equal(cents.Truncate(0)) {
		return fmt.Errorf("交易金额最小单位为0.01，当前金额精度过高")
	}

	return nil
}

// GetAmountDecimal 获取 decimal.Decimal 格式的金额
func (r *UserTradeRequestString) GetAmountDecimal() (decimal.Decimal, error) {
	// 尝试解码为字符串或数字
	var amountStr string
	if err := json.Unmarshal(r.Amount, &amountStr); err != nil {
		var amountFloat float64
		if err := json.Unmarshal(r.Amount, &amountFloat); err != nil {
			return decimal.Decimal{}, fmt.Errorf("无效的金额格式: %s", string(r.Amount))
		}
		amountStr = fmt.Sprintf("%.10f", amountFloat)
	}

	// 使用 decimal 解析金额
	amountDecimal, err := decimal.NewFromString(amountStr)
	if err != nil {
		return decimal.Decimal{}, fmt.Errorf("无效的金额格式: %s", amountStr)
	}
	return amountDecimal, nil
}

// UserTradeResponse 用户交易响应
type UserTradeResponse struct {
	Code      int         `json:"code"`
	Msg       string      `json:"msg"`
	RequestID string      `json:"requestId"`
	Data      interface{} `json:"data"`
}
