package data

import "time"

// BasinInfo 对应数据库中的 basin_info 表结构
type BasinInfo struct {
	ID              uint64    `gorm:"primaryKey"`
	BasinName       string    `gorm:"size:100;comment:流域名称"` // 流域名称
	ControlArea     float64   `gorm:"comment:控制面积(Km^2)"`    // 控制面积 (Km^2)
	MainStreamSlope float64   `gorm:"comment:干流坡度"`          // 干流坡度
	CreatedAt       time.Time // GORM 会自动处理
	UpdatedAt       time.Time // GORM 会自动处理
}

// TableName 指定 BasinInfo 模型对应的数据库表名
func (BasinInfo) TableName() string {
	return "basin_info"
}
