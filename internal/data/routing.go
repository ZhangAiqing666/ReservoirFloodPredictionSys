package data

import (
	// For potential future use
	routingV1 "ReservoirFloodPrediction/api/routing/v1"
	"ReservoirFloodPrediction/internal/biz"
	"context"
	"errors"
	"fmt"
	"strconv"

	"gorm.io/gorm"
)

// routingRepo 结构体实现了 biz.RoutingRepo 接口
type routingRepo struct {
	data *Data // 假设我们有 data.Data 结构，可能包含数据库连接等
	// 可以注入 logger 等
}

// NewRoutingRepo 创建一个新的 routingRepo
func NewRoutingRepo(data *Data) biz.RoutingRepo {
	return &routingRepo{
		data: data,
	}
}

// GetReservoirParams 从数据库获取水库参数
func (r *routingRepo) GetReservoirParams(ctx context.Context, reservoirID string) (*biz.Reservoir, error) {
	// 将传入的字符串 ID 转换为数据库模型所需的 uint64 类型
	// 确保这与 internal/data/reservoir.go 中 Reservoir 模型的 ID 类型匹配
	resID, err := strconv.ParseUint(reservoirID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid reservoir ID format '%s': %w", reservoirID, err)
	}

	// 1. 查询水库基本信息 (使用 reservoir.go 中定义的 Reservoir 模型)
	var reservoirModel Reservoir // <--- 使用 reservoir.go 中的 Reservoir 模型
	if err := r.data.db.WithContext(ctx).First(&reservoirModel, resID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("reservoir with ID '%s' not found in database: %w", reservoirID, err)
		}
		return nil, fmt.Errorf("failed to query reservoir basic info for ID %s: %w", reservoirID, err)
	}

	// 2. 查询库容曲线点，并按水位升序排序 (使用 reservoir.go 中定义的 StorageCurvePoint 模型)
	var storageCurveModels []StorageCurvePoint // <--- 使用 StorageCurvePoint 模型
	if err := r.data.db.WithContext(ctx).Where("reservoir_id = ?", resID).Order("level asc").Find(&storageCurveModels).Error; err != nil {
		return nil, fmt.Errorf("failed to query storage curve for reservoir ID %s: %w", reservoirID, err)
	}

	// 3. 查询泄流曲线点，并按水位升序排序 (使用 reservoir.go 中定义的 DischargeCurvePoint 模型)
	var dischargeCurveModels []DischargeCurvePoint // <--- 使用 DischargeCurvePoint 模型
	if err := r.data.db.WithContext(ctx).Where("reservoir_id = ?", resID).Order("level asc").Find(&dischargeCurveModels).Error; err != nil {
		return nil, fmt.Errorf("failed to query discharge curve for reservoir ID %s: %w", reservoirID, err)
	}

	// 4. 转换数据为 biz.Reservoir 结构
	bizRes := &biz.Reservoir{
		ID:   reservoirID,         // 业务层使用原始的字符串 ID
		Name: reservoirModel.Name, // 从查询到的模型获取名称
		Levels: &routingV1.CharacteristicLevels{
			// 从查询到的模型获取特征水位，使用辅助函数处理 nil
			FloodLimitWaterLevel: dereferenceFloat64(reservoirModel.FloodLimitWaterLevel),
			NormalWaterLevel:     dereferenceFloat64(reservoirModel.NormalWaterLevel),
			DesignFloodLevel:     dereferenceFloat64(reservoirModel.DesignFloodLevel),
			CheckFloodLevel:      dereferenceFloat64(reservoirModel.CheckFloodLevel),
		},
		// 从查询到的模型获取下游安全泄量
		DownstreamSafeDischarge: dereferenceFloat64(reservoirModel.DownstreamSafeDischarge),
		StorageCurve:            make([]*routingV1.CurvePoint, len(storageCurveModels)),
		DischargeCurve:          make([]*routingV1.CurvePoint, len(dischargeCurveModels)),
	}

	// 转换库容曲线点
	for i, model := range storageCurveModels {
		bizRes.StorageCurve[i] = &routingV1.CurvePoint{
			Level: model.Level,
			Value: model.Volume,
		}
	}

	// 转换泄流曲线点
	for i, model := range dischargeCurveModels {
		bizRes.DischargeCurve[i] = &routingV1.CurvePoint{
			Level: model.Level,
			Value: model.Discharge,
		}
	}

	return bizRes, nil
}

// dereferenceFloat64 辅助函数 (如果 routing.go 中没有，需要添加)
func dereferenceFloat64(f *float64) float64 {
	if f == nil {
		return 0 // 或者根据业务逻辑返回 math.NaN()
	}
	return *f
}
