package data

import (
	// 需要导入 biz 包以使用接口类型

	"ReservoirFloodPrediction/internal/conf"
	"errors"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(
	NewData,
	NewMapDataRepo,
	NewInflowRepo,
	NewRoutingRepo,
	NewUserRepo,
)

// Data .
type Data struct {
	db *gorm.DB
}

// newDB is an internal function used to create a database connection. It is not exported.
func newDB(c *conf.Data, logger log.Logger) (*gorm.DB, error) {
	helper := log.NewHelper(log.With(logger, "module", "data/gorm-connection"))

	if c == nil || c.Database == nil || c.Database.Source == "" {
		helper.Error("database configuration is missing or invalid")
		return nil, errors.New("database configuration is missing or invalid")
	}

	db, err := gorm.Open(mysql.Open(c.Database.Source), &gorm.Config{})
	if err != nil {
		helper.Errorf("failed opening connection to mysql: %v", err)
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		helper.Errorf("failed getting underlying sql.DB: %v", err)
		return nil, err // GORM v2 recommends returning directly on error without manual closing
	}
	if err := sqlDB.Ping(); err != nil {
		helper.Errorf("failed pinging database: %v", err)
		return nil, err // Same as above
	}

	helper.Info("mysql database connection established and verified")
	return db, nil
}

// NewData .
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	helper := log.NewHelper(log.With(logger, "module", "data/initialize"))

	// Initialize the database connection
	db, err := newDB(c, logger) // Call the internal function
	if err != nil {
		helper.Errorf("failed to initialize database: %v", err)
		return nil, nil, err // Initialize failed, return error directly
	}

	// !!! 确保 AutoMigrate 使用正确的模型 !!!
	err = db.AutoMigrate(
		&User{},                // 假设 User 定义在 user.go 或 data.go (无冲突)
		&BasinInfo{},           // 假设 BasinInfo 定义在 basin.go
		&Reservoir{},           // <--- 使用 reservoir.go 中定义的 Reservoir
		&StorageCurvePoint{},   // <--- 添加库容曲线模型
		&DischargeCurvePoint{}, // <--- 添加泄流曲线模型
	)
	if err != nil {
		helper.Errorf("failed to auto migrate database tables: %v", err)
		// 根据策略决定是否返回错误，开发阶段可以先只打印日志
		// return nil, nil, err
	}
	helper.Info("database auto migration checked/completed")

	d := &Data{
		db: db,
	}

	// Define the cleanup function used to close all data resources
	cleanup := func() {
		helper.Info("closing the data resources")
		sqlDB, err := db.DB()
		if err != nil {
			helper.Errorf("failed to get sql.DB for closing: %v", err)
			// Even if getting fails, log it, as it might not need to return immediately
		} else {
			if err := sqlDB.Close(); err != nil {
				helper.Errorf("failed to close database connection: %v", err)
			} else {
				helper.Info("database connection closed successfully")
			}
		}
		// If there are other resources (like Redis), add closing logic here
	}

	return d, cleanup, nil
}
