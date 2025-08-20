package service

import (
	"context"
	"fmt"
	"math"
	"one-pass/model"
	"time"

	"gorm.io/gorm"
)

// UserTrade 用户之间转账交易 - 高并发优化版本
func (s *Service) UserTrade(sourceUID, targetUID int64, amount float64) error {
	// 参数验证（简单检查，详细验证已在 handler 层完成）
	if sourceUID == targetUID {
		return fmt.Errorf("源账户和目标账户不能相同")
	}

	if amount <= 0 {
		return fmt.Errorf("交易金额必须大于0")
	}

	// 获取账户锁顺序 - 防止死锁
	firstUID, secondUID := sourceUID, targetUID
	if sourceUID > targetUID {
		firstUID, secondUID = targetUID, sourceUID
	}

	// 获取分布式锁
	lockKey1 := fmt.Sprintf("account_lock:%d", firstUID)
	lockKey2 := fmt.Sprintf("account_lock:%d", secondUID)

	ctx := context.Background()
	lockExpiry := 10 * time.Second

	// 按顺序获取锁，防止死锁
	lock1 := s.rdb.SetNX(ctx, lockKey1, "locked", lockExpiry)
	if !lock1.Val() {
		return fmt.Errorf("账户%d正在处理其他交易，请稍后重试", firstUID)
	}
	defer s.rdb.Del(ctx, lockKey1)

	if firstUID != secondUID {
		lock2 := s.rdb.SetNX(ctx, lockKey2, "locked", lockExpiry)
		if !lock2.Val() {
			return fmt.Errorf("账户%d正在处理其他交易，请稍后重试", secondUID)
		}
		defer s.rdb.Del(ctx, lockKey2)
	}

	// 使用数据库事务确保数据一致性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 使用行级锁查询源账户余额 - FOR UPDATE确保并发安全
		var sourceBalance model.UserBalance
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where("uid = ?", sourceUID).First(&sourceBalance).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("源账户不存在")
			}
			return fmt.Errorf("查询源账户余额失败: %v", err)
		}

		// 检查余额是否足够
		if sourceBalance.Amount < amount {
			return fmt.Errorf("源账户余额不足，当前余额: %.2f，需要: %.2f", sourceBalance.Amount, amount)
		}

		// 使用行级锁查询目标账户
		var targetBalance model.UserBalance
		err = tx.Set("gorm:query_option", "FOR UPDATE").Where("uid = ?", targetUID).First(&targetBalance).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return fmt.Errorf("目标账户不存在")
			}
			return fmt.Errorf("查询目标账户失败: %v", err)
		}

		// 计算新余额
		newSourceAmount := math.Round((sourceBalance.Amount-amount)*100) / 100
		newTargetAmount := math.Round((targetBalance.Amount+amount)*100) / 100

		// 批量更新 - 减少数据库交互次数
		err = tx.Model(&model.UserBalance{}).Where("uid IN (?)", []int64{sourceUID, targetUID}).
			Updates(map[string]interface{}{
				"amount": gorm.Expr("CASE WHEN uid = ? THEN ? WHEN uid = ? THEN ? END",
					sourceUID, newSourceAmount, targetUID, newTargetAmount),
			}).Error
		if err != nil {
			return fmt.Errorf("更新账户余额失败: %v", err)
		}

		// 更新Redis缓存，事务成功后执行
		// 如果Redis更新失败，事务会回滚
		err = s.updateUserBalanceInRedis(sourceUID, newSourceAmount)
		if err != nil {
			fmt.Printf("[WARNING] 更新源账户Redis缓存失败: %v\n", err)
			// 可以选择不返回错误，允许继续执行
		}

		err = s.updateUserBalanceInRedis(targetUID, newTargetAmount)
		if err != nil {
			fmt.Printf("[WARNING] 更新目标账户Redis缓存失败: %v\n", err)
			// 可以选择不返回错误，允许继续执行
		}

		fmt.Printf("[INFO] 交易完成: 源账户%d余额 %.2f -> %.2f, 目标账户%d余额 %.2f -> %.2f\n",
			sourceUID, sourceBalance.Amount, newSourceAmount,
			targetUID, targetBalance.Amount, newTargetAmount)

		return nil
	})
}
