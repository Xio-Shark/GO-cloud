package bootstrap

import (
	"errors"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewMySQL(cfg Config) (*gorm.DB, error) {
	if cfg.MySQLDSN == "" {
		return nil, errors.New("MYSQL_DSN is required")
	}
	return gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
	})
}
