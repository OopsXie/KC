package service

import (
	"context"
	"fmt"
	"math"
	"one-pass/config"
	"one-pass/model"
	"one-pass/utils"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	cfg *config.Config
	db  *gorm.DB
	rdb *redis.Client
}

// 全局配置变量 - 统一配置参数
const (
	// 基础配置
	MaxTransferAmount = 10000.0 // 单次转账最大金额

	// 并发控制参数 - 统一命名
	DefaultConcurrency   = 20 // 默认并发数
	MinConcurrency       = 3  // 最小并发数
	MaxConcurrency       = 60 // 最大并发数
	ConcurrencyThreshold = 90 // 成功率阈值（用于动态调整）

	// 超时配置
	FastPhaseTimeout = 30 // 快速阶段超时（秒）
)

func NewService(cfg *config.Config, db *gorm.DB, rdb *redis.Client) *Service {
	return &Service{cfg: cfg, db: db, rdb: rdb}
}

// Redis操作辅助函数
const (
	UserBalanceRedisKey = "user_balance:%d"     // 用户余额Redis键模板
	UserBalanceExpiry   = 24 * time.Hour        // 用户余额过期时间
	BatchPayRedisKey    = "batch_pay:%s:uid:%d" // 批量支付临时键模板
)

// 从Redis获取用户余额
func (s *Service) getUserBalanceFromRedis(uid int64) (float64, error) {
	key := fmt.Sprintf(UserBalanceRedisKey, uid)
	balanceStr, err := s.rdb.Get(context.Background(), key).Result()
	if err != nil {
		if err == redis.Nil {
			return 0, fmt.Errorf("用户余额不存在于Redis中")
		}
		return 0, fmt.Errorf("redis查询失败: %v", err)
	}

	var balance float64
	if _, err := fmt.Sscanf(balanceStr, "%f", &balance); err != nil {
		return 0, fmt.Errorf("解析余额失败: %v", err)
	}

	return balance, nil
}

// 将用户余额存储到Redis
func (s *Service) setUserBalanceToRedis(uid int64, balance float64) error {
	key := fmt.Sprintf(UserBalanceRedisKey, uid)
	balanceStr := fmt.Sprintf("%.2f", balance)

	return s.rdb.Set(context.Background(), key, balanceStr, UserBalanceExpiry).Err()
}

// 批量将用户余额存储到Redis
func (s *Service) batchSetUserBalancesToRedis(userBalances []model.UserBalance) error {
	pipe := s.rdb.Pipeline()

	for _, balance := range userBalances {
		key := fmt.Sprintf(UserBalanceRedisKey, balance.UID)
		balanceStr := fmt.Sprintf("%.2f", balance.Amount)
		pipe.Set(context.Background(), key, balanceStr, UserBalanceExpiry)
	}

	_, err := pipe.Exec(context.Background())
	return err
}

// 更新Redis中的用户余额
func (s *Service) updateUserBalanceInRedis(uid int64, newBalance float64) error {
	key := fmt.Sprintf(UserBalanceRedisKey, uid)
	balanceStr := fmt.Sprintf("%.2f", newBalance)

	// 使用SET操作更新余额，同时刷新过期时间
	return s.rdb.Set(context.Background(), key, balanceStr, UserBalanceExpiry).Err()
}

// 从Redis删除用户余额缓存
func (s *Service) deleteUserBalanceFromRedis(uid int64) error {
	key := fmt.Sprintf(UserBalanceRedisKey, uid)
	return s.rdb.Del(context.Background(), key).Err()
}

// 初始化Redis缓存，将数据库中的用户余额加载到Redis
func (s *Service) InitializeRedisCache() error {
	fmt.Println("[INFO] 开始初始化Redis缓存...")

	var userBalances []model.UserBalance
	err := s.db.Find(&userBalances).Error
	if err != nil {
		return fmt.Errorf("从数据库读取用户余额失败: %v", err)
	}

	if len(userBalances) == 0 {
		fmt.Println("[INFO] 数据库中没有用户余额数据")
		return nil
	}

	// 批量写入Redis
	err = s.batchSetUserBalancesToRedis(userBalances)
	if err != nil {
		return fmt.Errorf("批量写入Redis失败: %v", err)
	}

	fmt.Printf("[INFO] 成功初始化 %d 条用户余额到Redis缓存\n", len(userBalances))
	return nil
}

// 刷新指定用户的Redis缓存
func (s *Service) RefreshUserBalanceCache(uid int64) error {
	balance, err := s.getUserBalanceFromDB(uid)
	if err != nil {
		// 用户不存在于数据库，删除Redis缓存
		s.deleteUserBalanceFromRedis(uid)
		return fmt.Errorf("用户不存在于数据库中")
	}

	return s.setUserBalanceToRedis(uid, balance)
}

// 从数据库获取用户余额
func (s *Service) getUserBalanceFromDB(uid int64) (float64, error) {
	var userBalance model.UserBalance
	err := s.db.Where("uid = ?", uid).First(&userBalance).Error
	if err != nil {
		return 0, fmt.Errorf("数据库查询失败: %v", err)
	}

	return userBalance.Amount, nil
}

// 验证并修复用户余额差异
func (s *Service) ValidateAndFixUserBalance(uid int64, expectedAmount float64) error {
	// 从数据库获取当前余额
	currentBalance, err := s.getUserBalanceFromDB(uid)
	if err != nil {
		return fmt.Errorf("获取用户 %d 当前余额失败: %v", uid, err)
	}

	difference := currentBalance - expectedAmount
	if math.Abs(difference) > 0.01 { // 允许1分钱的精度误差
		fmt.Printf("[WARNING] 用户 %d 余额异常：当前 %.2f，期望 %.2f，差异 %.2f\n",
			uid, currentBalance, expectedAmount, difference)

		// 如果差异是10000的倍数，可能是最大金额转账丢失
		if math.Mod(math.Abs(difference), 10000.0) < 0.01 {
			lostTransfers := int(math.Abs(difference) / 10000.0)
			fmt.Printf("[ALERT] 用户 %d 可能丢失了 %d 次最大金额(10000.00)转账\n", uid, lostTransfers)
		}

		return fmt.Errorf("用户 %d 余额验证失败，差异: %.2f", uid, difference)
	}

	return nil
}

// 批量验证用户余额
func (s *Service) BatchValidateUserBalances(expectedBalances map[int64]float64) []error {
	var errors []error

	for uid, expectedAmount := range expectedBalances {
		if err := s.ValidateAndFixUserBalance(uid, expectedAmount); err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

// 辅助函数
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (s *Service) HandleBatchPay(batchId string, uids []int64) error {
	// // 第一步：调用批量支付开始接口
	// success, err := utils.CallBatchPayBegin(s.cfg, batchId, uids)
	// if err != nil || !success {
	// 	fmt.Printf("[ERROR] 批量支付开始接口调用失败: %v\n", err)
	// 	return fmt.Errorf("批量支付开始接口调用失败: %v", err)
	// }

	fmt.Printf("[INFO] 开始处理批量支付，批次ID: %s，用户数量: %d\n", batchId, len(uids))

	// 并发处理每个用户的充值 - 为每个用户启动独立协程
	var wg sync.WaitGroup
	for _, uid := range uids {
		wg.Add(1)
		go func(uid int64) {
			defer wg.Done()

			// 使用高效二分法充值，并将结果存储到Redis
			total := s.efficientChargeAllBalance(uid, batchId)

			// 将累计金额存储到Redis - 增加重试机制
			redisKey := fmt.Sprintf("batch_pay:%s:uid:%d", batchId, uid)

			// 重试存储到Redis，最多重试3次
			var redisErr error
			for retry := 0; retry < 3; retry++ {
				redisErr = s.rdb.Set(context.Background(), redisKey, total, 24*time.Hour).Err()
				if redisErr == nil {
					//fmt.Printf("[INFO] 用户 %d 充值完成，总金额: %.2f，已存储到Redis (重试次数: %d)\n", uid, total, retry)
					break
				}
				fmt.Printf("[WARNING] 用户 %d Redis存储失败，重试 %d/3: %v\n", uid, retry+1, redisErr)
				time.Sleep(time.Duration(retry+1) * 100 * time.Millisecond) // 递增延迟
			}

			if redisErr != nil {
				fmt.Printf("[ERROR] 用户 %d Redis存储最终失败: %v\n", uid, redisErr)
			}
		}(uid)
	}
	wg.Wait()

	fmt.Printf("[INFO] 所有用户充值处理完成，开始批量写入数据库\n")

	// 批量从Redis读取数据并写入数据库
	err := s.batchWriteToDatabase(batchId, uids)
	if err != nil {
		fmt.Printf("[ERROR] 批量写入数据库失败: %v\n", err)
		return fmt.Errorf("批量写入数据库失败: %v", err)
	}

	fmt.Printf("[INFO] 调用批量支付完成接口\n")

	// 调用批量支付完成接口
	success, err := utils.CallBatchPayFinish(s.cfg, batchId)
	if err != nil || !success {
		fmt.Printf("[ERROR] 批量支付完成接口调用失败: %v\n", err)
		return fmt.Errorf("批量支付完成接口调用失败: %v", err)
	}
	fmt.Printf("[INFO] 批量支付完成接口调用成功\n")
	fmt.Printf("[INFO] 批量支付处理完成\n")

	return nil
}

// 高效充值所有余额 - 优化连接管理版本（防止端口耗尽）
func (s *Service) efficientChargeAllBalance(uid int64, batchId string) float64 {
	fmt.Printf("[INFO] 开始为用户 %d 执行高效充值流程\n", uid)
	total := 0.0

	// 第一阶段：快速消耗阶段 - 使用统一的配置参数
	phase1Total := s.aggressiveFastDrainPhase(uid, batchId, MaxTransferAmount, DefaultConcurrency, FastPhaseTimeout, MaxConcurrency)
	total += phase1Total
	//fmt.Printf("[INFO] 用户 %d 第一阶段完成，转账金额: %.2f\n", uid, phase1Total)

	// 第二阶段：并行二分法处理剩余资金（保持不变）
	phase2Total := s.parallelPrecisionDrainPhase(uid, batchId, MaxTransferAmount)
	total += phase2Total
	//fmt.Printf("[INFO] 用户 %d 第二阶段完成，转账金额: %.2f\n", uid, phase2Total)

	// 第三阶段：最终清理阶段（保持不变）
	phase3Total := s.finalCleanupPhase(uid, batchId)
	total += phase3Total
	//fmt.Printf("[INFO] 用户 %d 第三阶段完成，转账金额: %.2f\n", uid, phase3Total)

	// 第四阶段：彻底清理阶段（保持不变）
	phase4Total := s.thoroughCleanupPhase(uid, batchId)
	total += phase4Total
	//fmt.Printf("[INFO] 用户 %d 第四阶段完成，转账金额: %.2f\n", uid, phase4Total)

	total = math.Round(total*100) / 100
	// fmt.Printf("[INFO] 用户 %d 高效充值完成，总金额: %.2f（阶段明细：%.2f + %.2f + %.2f + %.2f）\n",
	// 	uid, total, phase1Total, phase2Total, phase3Total, phase4Total)
	return total
}

func (s *Service) aggressiveFastDrainPhase(uid int64, batchId string, maxAmount float64, initialConcurrency, timeoutSeconds, maxConcurrencyLimit int) float64 {
	fmt.Printf("[INFO] 用户 %d 开始第一阶段：快速消耗阶段，最大金额: %.2f\n", uid, maxAmount)

	total := 0.0
	currentConcurrency := initialConcurrency

	// 用于动态调整并发数的统计
	var successCount, totalRequests int64

	// 使用更大的信号量和结果缓冲
	semaphore := make(chan struct{}, currentConcurrency)
	results := make(chan float64, 5000) // 大账户需要更大的缓冲
	var wg sync.WaitGroup
	var mu sync.Mutex
	var transferCount int
	var consecutiveFailures int
	var maxAmountSuccessCount int // 统计最大金额成功次数

	// 控制循环退出
	done := make(chan bool, 1)

	// 更激进的动态调整策略
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond) // 每500ms调整一次，更频繁
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				if totalRequests > 10 { // 更少样本就开始调整
					successRate := float64(successCount) / float64(totalRequests) * 100

					if successRate > 80 && currentConcurrency < maxConcurrencyLimit {
						// 成功率高，大幅增加并发
						increase := min(10, maxConcurrencyLimit-currentConcurrency)
						currentConcurrency += increase

						// 扩大信号量
						for i := 0; i < increase; i++ {
							select {
							case semaphore <- struct{}{}:
							default:
							}
						}
					} else if successRate < 20 && currentConcurrency > 10 {
						// 成功率很低，减少并发
						currentConcurrency = max(currentConcurrency-10, 7)
					}

					// 重置统计
					successCount, totalRequests = 0, 0
				}
				mu.Unlock()
			}
		}
	}()

	// 启动更多工作协程
	for i := 0; i < currentConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// 获取信号量
					semaphore <- struct{}{}

					transactionId := uuid.NewString()
					ok, err := s.concurrentCallPay(transactionId, uid, maxAmount)

					<-semaphore // 释放信号量

					// 更新统计
					mu.Lock()
					totalRequests++
					if err == nil && ok {
						successCount++
					}
					mu.Unlock()

					if err != nil {
						mu.Lock()
						consecutiveFailures++
						if consecutiveFailures >= currentConcurrency*5 { // 更宽松的网络错误容忍
							mu.Unlock()
							return
						}
						mu.Unlock()
						continue
					}

					if ok {
						results <- maxAmount
						mu.Lock()
						consecutiveFailures = 0
						maxAmountSuccessCount++ // 统计最大金额成功次数
						// 添加成功转账的调试日志
						//fmt.Printf("[DEBUG] 用户 %d 工作线程 %d 成功转账: %.2f (第%d次最大金额转账)\n", uid, workerID, maxAmount, maxAmountSuccessCount)
						mu.Unlock()
					} else {
						mu.Lock()
						consecutiveFailures++
						if consecutiveFailures >= currentConcurrency*2 { // 大账户可能需要更多尝试
							mu.Unlock()
							return
						}
						mu.Unlock()
					}
				}
			}
		}(i)
	}

	// 收集结果协程
	resultDone := make(chan bool)
	go func() {
		for amount := range results {
			mu.Lock()
			total += amount
			transferCount++
			mu.Unlock()
		}
		resultDone <- true
	}()

	// 更长的超时时间
	timeout := time.Duration(timeoutSeconds) * time.Second
	<-time.After(timeout)

	close(done)
	wg.Wait()
	close(results)
	<-resultDone

	// fmt.Printf("[INFO] 用户 %d 第一阶段完成 - 总转账: %.2f, 最大金额(%.2f)成功次数: %d, 总转账次数: %d\n",
	// 	uid, total, maxAmount, maxAmountSuccessCount, transferCount)

	return total
}

// 并行二分法处理
func (s *Service) parallelPrecisionDrainPhase(uid int64, batchId string, maxAmount float64) float64 {
	total := 0.0

	// 将金额范围分成多段并行处理
	ranges := []struct{ low, high float64 }{
		{0.01, 1000.0},
		{1000.01, 5000.0},
		{5000.01, 10000.0},
	}

	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, r := range ranges {
		wg.Add(1)
		go func(low, high float64) {
			defer wg.Done()
			rangeTotal := s.binarySearchInRange(uid, batchId, low, high)
			mu.Lock()
			total += rangeTotal
			mu.Unlock()
		}(r.low, r.high)
	}

	wg.Wait()
	return total
}

// 在指定范围内进行二分查找
func (s *Service) binarySearchInRange(uid int64, batchId string, lowLimit, highLimit float64) float64 {
	low := lowLimit
	high := highLimit
	totalTransferred := 0.0

	maxIterations := 10 // 减少迭代次数，提高效率
	for iteration := 0; iteration < maxIterations && high-low > 1.0; iteration++ {
		mid := math.Round((low+high)/2*100) / 100

		transactionId := uuid.NewString()
		ok, err := s.concurrentCallPay(transactionId, uid, mid)
		if err != nil {
			break
		}

		if ok {
			consecutiveSuccesses := 1
			totalTransferred += mid

			// 快速连续转账，限制次数避免过长时间
			maxContinuous := 500 // 减少连续转账次数
			for consecutiveSuccesses < maxContinuous {
				transactionId = uuid.NewString()
				ok, err = s.concurrentCallPay(transactionId, uid, mid)
				if err != nil || !ok {
					break
				}
				totalTransferred += mid
				consecutiveSuccesses++
			}

			low = mid + 1.0
		} else {
			high = mid - 1.0
		}
	}

	return totalTransferred
}

// 第一阶段：快速消耗阶段 - 连续转账最大金额（智能并发调整）
func (s *Service) fastDrainPhase(uid int64, batchId string, maxAmount float64) float64 {
	total := 0.0
	currentConcurrency := DefaultConcurrency

	// 用于动态调整并发数的统计
	var successCount, totalRequests int64

	// 使用信号量控制并发数
	semaphore := make(chan struct{}, currentConcurrency)
	results := make(chan float64, 1000)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var transferCount int
	var consecutiveFailures int

	// 控制循环退出
	done := make(chan bool, 1)

	// 动态调整并发数的协程
	go func() {
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				mu.Lock()
				if totalRequests > 20 { // 有足够样本时才调整
					successRate := float64(successCount) / float64(totalRequests) * 100

					if successRate > float64(ConcurrencyThreshold) && currentConcurrency < MaxConcurrency {
						// 成功率高，增加并发
						currentConcurrency = min(currentConcurrency+5, MaxConcurrency)

						// 扩大信号量
						for i := 0; i < 5; i++ {
							select {
							case semaphore <- struct{}{}:
							default:
							}
						}
					} else if successRate < 30 && currentConcurrency > MinConcurrency {
						// 成功率低，减少并发
						currentConcurrency = max(currentConcurrency-3, MinConcurrency)
					}

					// 重置统计
					successCount, totalRequests = 0, 0
				}
				mu.Unlock()
			}
		}
	}()

	// 启动工作协程
	for i := 0; i < currentConcurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for {
				select {
				case <-done:
					return
				default:
					// 获取信号量
					semaphore <- struct{}{}

					transactionId := uuid.NewString()
					ok, err := s.concurrentCallPay(transactionId, uid, maxAmount)

					<-semaphore // 释放信号量

					// 更新统计
					mu.Lock()
					totalRequests++
					if err == nil && ok {
						successCount++
					}
					mu.Unlock()

					if err != nil {
						mu.Lock()
						consecutiveFailures++
						if consecutiveFailures >= currentConcurrency*3 {
							mu.Unlock()
							return // 网络错误太多，退出
						}
						mu.Unlock()
						continue
					}

					if ok {
						results <- maxAmount
						mu.Lock()
						consecutiveFailures = 0 // 重置失败计数
						mu.Unlock()
					} else {
						// 转账失败，可能余额不足
						mu.Lock()
						consecutiveFailures++
						if consecutiveFailures >= currentConcurrency {
							mu.Unlock()
							return // 连续失败太多，可能余额不足
						}
						mu.Unlock()
					}
				}
			}
		}(i)
	}

	// 收集结果协程
	resultDone := make(chan bool)
	go func() {
		for amount := range results {
			mu.Lock()
			total += amount
			transferCount++
			mu.Unlock()
		}
		resultDone <- true
	}()

	// 动态等待，最多等待配置的时间
	timeout := time.Duration(FastPhaseTimeout) * time.Second
	select {
	case <-time.After(timeout):
		// 超时退出
	case <-time.After(100 * time.Millisecond):
		// 检查是否所有工作线程都因为余额不足而退出
		mu.Lock()
		if consecutiveFailures >= currentConcurrency {
			mu.Unlock()
			close(done)
		} else {
			mu.Unlock()
			// 继续等待
			<-time.After(timeout - 100*time.Millisecond)
		}
	}

	close(done)
	wg.Wait()
	close(results)
	<-resultDone

	return total
}

// 第二阶段：精确处理阶段 - 二分法处理剩余资金
func (s *Service) precisionDrainPhase(uid int64, batchId string, maxAmount float64) float64 {
	total := 0.0

	// 在接口限制范围内进行二分查找
	total += s.binarySearchMaxAmount(uid, batchId, maxAmount, 0)

	return total
}

// 二分查找最大可转账金额 - 遵循接口限制
func (s *Service) binarySearchMaxAmount(uid int64, batchId string, maxLimit float64, workerID int) float64 {
	low := 0.01
	high := math.Min(maxLimit, MaxTransferAmount) // 确保不超过接口限制
	totalTransferred := 0.0

	// 减少二分查找次数，提高效率
	maxIterations := 15                                                            // 从20减少到15
	for iteration := 0; iteration < maxIterations && high-low > 0.1; iteration++ { // 精度从0.01放宽到0.1
		mid := math.Round((low+high)/2*100) / 100

		transactionId := uuid.NewString()

		ok, err := s.concurrentCallPay(transactionId, uid, mid)
		if err != nil {
			break
		}

		if ok {
			// 成功后，连续尝试相同金额，榨干该金额级别的所有资金
			consecutiveSuccesses := 1
			totalTransferred += mid

			// 连续转账相同金额直到失败，限制最大次数
			maxContinuous := 1000 // 限制连续转账次数
			for consecutiveSuccesses < maxContinuous {
				transactionId = uuid.NewString()
				ok, err = s.concurrentCallPay(transactionId, uid, mid)
				if err != nil || !ok {
					break
				}
				totalTransferred += mid
				consecutiveSuccesses++
			}

			low = mid + 0.1 // 增大步长
		} else {
			high = mid - 0.1 // 增大步长
		}
	}

	return totalTransferred
}

// 并发支付调用，优化网络请求
func (s *Service) concurrentCallPay(transactionId string, uid int64, amount float64) (bool, error) {
	// 使用更短的超时时间，快速失败
	ok, err := utils.CallPay(s.cfg, transactionId, uid, amount)
	return ok, err
}

// 最终清理阶段：处理剩余零散金额 - 遵循接口限制
func (s *Service) finalCleanupPhase(uid int64, batchId string) float64 {
	total := 0.0

	// 简化金额数组，减少不必要的尝试
	amounts := []float64{9999.0, 5000.0, 1000.0, 500.0, 100.0, 50.0, 10.0, 5.0, 1.0, 0.1, 0.01}

	for _, amount := range amounts {
		consecutiveSuccesses := 0
		consecutiveFailures := 0
		maxFailures := 2 // 减少失败次数，更快跳转

		for consecutiveFailures < maxFailures {
			transactionId := uuid.NewString()

			ok, err := s.concurrentCallPay(transactionId, uid, amount)
			if err != nil {
				consecutiveFailures++
				continue
			}

			if ok {
				total += amount
				consecutiveSuccesses++
				consecutiveFailures = 0 // 重置失败计数
			} else {
				consecutiveFailures++
			}
		}
	}

	return total
}

// 第四阶段：彻底清理阶段 - 确保完全榨干账户
func (s *Service) thoroughCleanupPhase(uid int64, batchId string) float64 {
	total := 0.0

	// 大幅简化金额数组，只保留关键金额
	amounts := []float64{
		999.99, 500.0, 100.0, 50.0, 10.0, 5.0, 1.0, 0.5, 0.1, 0.05, 0.01,
	}

	for _, amount := range amounts {
		amount = math.Round(amount*100) / 100 // 确保精度
		consecutiveFailures := 0
		maxFailures := 1 // 只尝试1次，失败就跳过

		for consecutiveFailures < maxFailures {
			transactionId := uuid.NewString()

			ok, err := s.concurrentCallPay(transactionId, uid, amount)
			if err != nil {
				consecutiveFailures++
				continue
			}

			if ok {
				total += amount
				consecutiveFailures = 0 // 重置失败计数
			} else {
				consecutiveFailures++
			}
		}
	}

	// if total > 0 {
	// 	// fmt.Printf("[INFO] 用户 %d 彻底清理完成，转账: %.2f\n", uid, total)
	// } else {
	// 	fmt.Printf("[INFO] 用户 %d 彻底清理完成，账户已完全清空\n", uid)
	// }

	return total
}

// 批量写入数据库
func (s *Service) batchWriteToDatabase(batchId string, uids []int64) error {
	var userBalances []model.UserBalance

	// 从Redis批量读取数据
	for _, uid := range uids {
		redisKey := fmt.Sprintf("batch_pay:%s:uid:%d", batchId, uid)
		totalStr, err := s.rdb.Get(context.Background(), redisKey).Result()
		if err != nil {
			fmt.Printf("[ERROR] 从Redis读取用户 %d 数据失败: %v\n", uid, err)
			continue
		}

		var total float64
		if _, err := fmt.Sscanf(totalStr, "%f", &total); err != nil {
			fmt.Printf("[ERROR] 解析用户 %d 金额失败: %v\n", uid, err)
			continue
		}

		userBalances = append(userBalances, model.UserBalance{
			UID:    uid,
			Amount: math.Round(total*100) / 100,
		})
	}

	// 批量写入数据库
	if len(userBalances) > 0 {
		err := s.db.Clauses(clause.OnConflict{UpdateAll: true}).CreateInBatches(userBalances, 100).Error
		if err != nil {
			return fmt.Errorf("批量写入数据库失败: %v", err)
		}
		fmt.Printf("[INFO] 成功批量写入 %d 条用户余额记录\n", len(userBalances))

		// 批量更新Redis缓存 - 将用户余额持久化到Redis
		err = s.batchSetUserBalancesToRedis(userBalances)
		if err != nil {
			fmt.Printf("[WARNING] 批量更新Redis缓存失败: %v\n", err)
			// 注意：这里不返回错误，因为数据库写入已成功，Redis更新失败不应该影响主流程
		} else {
			fmt.Printf("[INFO] 成功批量更新 %d 条用户余额到Redis缓存\n", len(userBalances))
		}

		// 清理临时Redis缓存
		for _, uid := range uids {
			redisKey := fmt.Sprintf("batch_pay:%s:uid:%d", batchId, uid)
			s.rdb.Del(context.Background(), redisKey)
		}
	}

	return nil
}
