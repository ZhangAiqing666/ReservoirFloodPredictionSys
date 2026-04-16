package data

// "ReservoirFloodPrediction/internal/conf" // NewData 可能在 data.go 中定义，这里不需要 conf

// ProviderSet 定义移至 data.go
// var ProviderSet = wire.NewSet(...)

// 其他 data 包内的 wire 配置或绑定可以保留在这里（如果需要）

// 注意：Data 结构体 和 NewData 函数 应该在 data.go 或其他 data_*.go 文件中唯一定义。
// 注意：NewMapDataRepo, NewInflowRepo, NewRoutingRepo 等构造函数也需要在 data 包的文件中定义。
