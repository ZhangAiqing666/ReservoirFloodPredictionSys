package biz

import (
	"errors"
)

// 此文件通常为空或只包含包文档
// ProviderSet 定义移至 wire.go

// --- 共享错误定义 ---
var (
	ErrBasinInfoNotFound = errors.New("basin info not found")
	// 可以添加其他共享错误
)

// --- 共享接口定义 ---

// MapDataRepo 定义已移至 internal/biz/mapdata.go

// InflowRepo 定义了入库洪水分析数据访问需要实现的方法
// 目前可能为空，或者包含未来可能需要的如保存/加载历史数据的方法
type InflowRepo interface {
	// SaveSimulationData(ctx context.Context, data ...) error // 示例
	// GetSimulatedRainfall(ctx context.Context, id string) (...) error // 示例
}

// 注意：ProviderSet 定义在 data.go 中， wire.go 文件用于 biz 包的 ProviderSet
// 我们需要确保 biz/wire.go 中引用了正确的 UseCase 构造函数
