package healthcheck

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type Checker func(ctx context.Context) error

func NewDependencyChecker(db *gorm.DB, rdb *goredis.Client) Checker {
	return func(ctx context.Context) error {
		if db != nil {
			sqlDB, err := db.DB()
			if err != nil {
				return err
			}
			if err := sqlDB.PingContext(ctx); err != nil {
				return err
			}
		}
		if rdb != nil {
			if err := rdb.Ping(ctx).Err(); err != nil {
				return err
			}
		}
		return nil
	}
}

func CheckDependencies(ctx context.Context, db *gorm.DB, rdb *goredis.Client) error {
	return NewDependencyChecker(db, rdb)(ctx)
}
