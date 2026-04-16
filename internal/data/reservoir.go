package data

import "time"

// Reservoir 对应数据库中的 reservoirs 表结构
// 同时包含作为雨量站的属性
type Reservoir struct {
	ID          uint64  `gorm:"primaryKey"`
	Name        string  `gorm:"uniqueIndex;size:100;comment:水库/雨量站名称"` // 水库/雨量站名称，唯一
	Latitude    float64 `gorm:"comment:纬度"`                            // 纬度
	Longitude   float64 `gorm:"comment:经度"`                            // 经度
	ControlArea float64 `gorm:"comment:作为雨量站的控制面积(Km^2)"`              // 作为雨量站的控制面积 (Km^2)
	Weight      float64 `gorm:"comment:作为雨量站的权重"`                      // 作为雨量站的权重

	// --- 添加以下用于调洪计算的字段 ---
	FloodLimitWaterLevel    *float64 `gorm:"comment:汛限水位(m)"` // 使用指针以允许 NULL
	NormalWaterLevel        *float64 `gorm:"comment:正常蓄水位(m)"`
	DesignFloodLevel        *float64 `gorm:"comment:设计洪水位(m)"`
	CheckFloodLevel         *float64 `gorm:"comment:校核洪水位(m)"`
	DownstreamSafeDischarge *float64 `gorm:"comment:下游安全泄量(m³/s)"`
	// --- 结束添加 ---

	CreatedAt time.Time // GORM 自动处理
	UpdatedAt time.Time // GORM 自动处理
}

// TableName 指定 Reservoir 模型对应的数据库表名
func (Reservoir) TableName() string {
	return "reservoirs"
}

// --- 添加以下曲线模型 ---

// StorageCurvePoint 对应数据库中的 storage_curves 表
type StorageCurvePoint struct {
	ID          uint64  `gorm:"primaryKey"`
	ReservoirID uint64  `gorm:"index;not null"` // 外键, 确保类型与 Reservoir.ID 匹配
	Level       float64 `gorm:"not null"`
	Volume      float64 `gorm:"not null"` // 库容 (万m³)
	// 可以添加关联，如果需要
	// Reservoir    Reservoir `gorm:"foreignKey:ReservoirID"`
}

// TableName 指定表名
func (StorageCurvePoint) TableName() string {
	return "storage_curves" // 确保这是您的实际表名
}

// DischargeCurvePoint 对应数据库中的 discharge_curves 表
type DischargeCurvePoint struct {
	ID          uint64  `gorm:"primaryKey"`
	ReservoirID uint64  `gorm:"index;not null"` // 外键, 确保类型与 Reservoir.ID 匹配
	Level       float64 `gorm:"not null"`
	Discharge   float64 `gorm:"not null"` // 下泄流量 (m³/s)
	// Reservoir    Reservoir `gorm:"foreignKey:ReservoirID"`
}

// TableName 指定表名
func (DischargeCurvePoint) TableName() string {
	return "discharge_curves" // 确保这是您的实际表名
}

// --- 结束添加 ---
