package service

import (
	"fmt"
	"one-pass/model"
)

// QueryUserAmounts 查询用户余额 - 用于转账场景，优先从Redis读取
func (s *Service) QueryUserAmounts(uids []int64) ([]model.UserAmountData, error) {
	var result []model.UserAmountData

	// 逐个查询用户余额（转账场景使用Redis优先策略）
	for _, uid := range uids {
		balance, err := s.getUserBalanceFromRedis(uid)
		if err != nil {
			// Redis中不存在，尝试从数据库读取
			if err.Error() == "用户余额不存在于Redis中" {
				dbBalance, dbErr := s.getUserBalanceFromDB(uid)
				if dbErr != nil {
					// 数据库中也不存在
					result = append(result, model.UserAmountData{
						UID: uid,
						Msg: "用户不存在",
					})
					continue
				}

				// 从数据库读取成功后，同步到Redis
				syncErr := s.setUserBalanceToRedis(uid, dbBalance)
				if syncErr != nil {
					fmt.Printf("[WARNING] 同步用户 %d 余额到Redis失败: %v\n", uid, syncErr)
				}

				result = append(result, model.UserAmountData{
					UID:    uid,
					Amount: dbBalance,
				})
			} else {
				// Redis查询失败时，返回错误信息
				result = append(result, model.UserAmountData{
					UID: uid,
					Msg: fmt.Sprintf("查询失败: %v", err),
				})
			}
			continue
		}

		// Redis查询成功，直接返回余额
		result = append(result, model.UserAmountData{
			UID:    uid,
			Amount: balance,
		})
	}

	return result, nil
}

// QueryUserAmountsFromDB 查询用户余额 - 直接从数据库读取（兼容原有接口）
func (s *Service) QueryUserAmountsFromDB(uids []int64) ([]model.UserAmountData, error) {
	var userBalances []model.UserBalance

	// 从数据库查询用户余额
	err := s.db.Where("uid IN ?", uids).Find(&userBalances).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户余额失败: %v", err)
	}

	// 创建一个map来快速查找用户余额
	balanceMap := make(map[int64]float64)
	for _, balance := range userBalances {
		balanceMap[balance.UID] = balance.Amount
	}

	// 构造响应数据，区分存在和不存在的用户
	var result []model.UserAmountData
	for _, uid := range uids {
		if amount, exists := balanceMap[uid]; exists {
			// 用户存在，返回余额
			result = append(result, model.UserAmountData{
				UID:    uid,
				Amount: amount,
			})
		} else {
			// 用户不存在，返回错误信息
			result = append(result, model.UserAmountData{
				UID: uid,
				Msg: "用户不存在",
			})
		}
	}

	return result, nil
}
