package config

import (
	"log"
	"time"

	"one-pass/model"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	Kingstar struct {
		ID    string
		Token string
	}
	Server struct {
		IP      string
		BaseURL string
	}
	GitLab struct {
		URL string
	}
	API struct {
		PayURL            string
		BatchPayBeginURL  string
		BatchPayFinishURL string
	}
	MySQL struct {
		DSN string
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
}

func Load() *Config {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("读取配置失败: %v", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		log.Fatalf("配置解析失败: %v", err)
	}
	return &cfg
}

func InitDB(cfg *Config) *gorm.DB {
	var db *gorm.DB
	var err error

	maxRetries := 10
	for i := 1; i <= maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(cfg.MySQL.DSN), &gorm.Config{})
		if err == nil {
			break
		}
		log.Printf("数据库连接失败（第 %d 次重试）: %v", i, err)
		time.Sleep(3 * time.Second)
	}

	if err != nil {
		log.Fatalf("无法连接数据库: %v", err)
	}

	// 获取底层数据库连接并配置连接池
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}

	// 高并发优化配置
	sqlDB.SetMaxOpenConns(100)    // 最大打开连接数
	sqlDB.SetMaxIdleConns(20)     // 最大空闲连接数
	sqlDB.SetConnMaxLifetime(300) // 连接最大生存时间(秒)
	sqlDB.SetConnMaxIdleTime(60)  // 连接最大空闲时间(秒)

	if err := db.AutoMigrate(&model.UserBalance{}); err != nil {
		log.Fatalf("数据库迁移失败: %v", err)
	}

	return db
}

func InitRedis(cfg *Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
}
