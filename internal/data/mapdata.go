package data

import (
	"ReservoirFloodPrediction/internal/biz"
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"gorm.io/gorm"
)

// mapDataRepo 结构体实现了 biz.MapDataRepo 接口
type mapDataRepo struct {
	data *Data
	log  *log.Helper
}

// NewMapDataRepo 创建一个新的 mapDataRepo
func NewMapDataRepo(data *Data, logger log.Logger) biz.MapDataRepo {
	return &mapDataRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/mapdata")),
	}
}

// ListBasins 从数据库获取流域列表
func (r *mapDataRepo) ListBasins(ctx context.Context) ([]*biz.BasinInfo, error) {
	var basins []BasinInfo // 使用 data 包内定义的模型（假设存在且与 biz 匹配）
	if err := r.data.db.WithContext(ctx).Find(&basins).Error; err != nil {
		r.log.Errorf("failed to list basins: %v", err)
		return nil, err
	}

	bizBasins := make([]*biz.BasinInfo, len(basins))
	for i, b := range basins {
		bizBasins[i] = &biz.BasinInfo{
			ID:              b.ID,
			BasinName:       b.BasinName,
			ControlArea:     b.ControlArea,
			MainStreamSlope: b.MainStreamSlope,
		}
	}
	return bizBasins, nil
}

// ListReservoirs 从数据库获取水库/雨量站列表
func (r *mapDataRepo) ListReservoirs(ctx context.Context) ([]*biz.ReservoirInfo, error) {
	var reservoirs []Reservoir // 使用 data 包内定义的模型
	if err := r.data.db.WithContext(ctx).Find(&reservoirs).Error; err != nil {
		r.log.Errorf("failed to list reservoirs: %v", err)
		return nil, err
	}

	bizReservoirs := make([]*biz.ReservoirInfo, len(reservoirs))
	for i, res := range reservoirs {
		bizReservoirs[i] = &biz.ReservoirInfo{
			ID:          res.ID,
			Name:        res.Name,
			Latitude:    res.Latitude,
			Longitude:   res.Longitude,
			ControlArea: res.ControlArea,
			Weight:      res.Weight,
		}
	}
	return bizReservoirs, nil
}

// --- 新增 Repo 方法实现 ---

// GenerateAndSaveCurves 生成模拟曲线并保存到数据库
// 注意：这是一个非常简化的示例逻辑！
func (r *mapDataRepo) GenerateAndSaveCurves(ctx context.Context, reservoirID uint64) error {
	// 1. 检查水库是否存在 (可选，但推荐)
	var reservoir Reservoir
	if err := r.data.db.WithContext(ctx).First(&reservoir, reservoirID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("cannot generate curves, reservoir with ID %d not found", reservoirID)
		}
		return fmt.Errorf("failed to check reservoir existence for ID %d: %w", reservoirID, err)
	}

	// 2. 定义模拟参数 (!!! 这些应该基于实际水库特性 !!!)
	minLevel := dereferenceFloat64(reservoir.FloodLimitWaterLevel) // 尝试使用汛限水位作为下限
	maxLevel := dereferenceFloat64(reservoir.CheckFloodLevel)      // 尝试使用校核洪水位作为上限
	if minLevel == 0 {
		minLevel = 100
	} // 如果没有特征水位，使用默认值
	if maxLevel == 0 {
		maxLevel = 120
	}
	if minLevel >= maxLevel {
		maxLevel = minLevel + 10
	} // 确保有范围
	numPoints := 10 // 生成的点数
	minStorage := 100.0
	maxStorageFactor := 50.0 // 库容增长因子
	minDischarge := 0.0
	maxDischargeFactor := 10.0 // 泄量增长因子

	storagePoints := make([]StorageCurvePoint, numPoints)
	dischargePoints := make([]DischargeCurvePoint, numPoints)

	rand.Seed(time.Now().UnixNano()) // 初始化随机种子

	// 3. 生成模拟点 (非线性增长示例)
	for i := 0; i < numPoints; i++ {
		level := minLevel + (maxLevel-minLevel)*float64(i)/float64(numPoints-1)
		// 模拟库容 (指数增长)
		storage := minStorage + math.Pow(float64(i)/float64(numPoints-1), 2)*maxStorageFactor*(maxLevel-minLevel)*(1+rand.Float64()*0.1-0.05) // 添加随机扰动
		// 模拟泄量 (指数增长)
		discharge := minDischarge + math.Pow(float64(i)/float64(numPoints-1), 3)*maxDischargeFactor*(maxLevel-minLevel)*(1+rand.Float64()*0.1-0.05) // 添加随机扰动

		storagePoints[i] = StorageCurvePoint{
			ReservoirID: reservoirID,
			Level:       level,
			Volume:      storage,
		}
		dischargePoints[i] = DischargeCurvePoint{
			ReservoirID: reservoirID,
			Level:       level,
			Discharge:   discharge,
		}
	}

	// 4. 在事务中保存数据：先删除旧曲线，再插入新曲线
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除旧数据
		if err := tx.Where("reservoir_id = ?", reservoirID).Delete(&StorageCurvePoint{}).Error; err != nil {
			return fmt.Errorf("failed to delete old storage curve for reservoir ID %d: %w", reservoirID, err)
		}
		if err := tx.Where("reservoir_id = ?", reservoirID).Delete(&DischargeCurvePoint{}).Error; err != nil {
			return fmt.Errorf("failed to delete old discharge curve for reservoir ID %d: %w", reservoirID, err)
		}

		// 插入新数据
		if err := tx.Create(&storagePoints).Error; err != nil {
			return fmt.Errorf("failed to insert new storage curve for reservoir ID %d: %w", reservoirID, err)
		}
		if err := tx.Create(&dischargePoints).Error; err != nil {
			return fmt.Errorf("failed to insert new discharge curve for reservoir ID %d: %w", reservoirID, err)
		}

		return nil // 事务成功
	})

	if err != nil {
		r.log.Errorf("GenerateAndSaveCurves failed for reservoir %d: %v", reservoirID, err)
		return err
	}

	r.log.Infof("Successfully generated and saved curves for reservoir %d", reservoirID)
	return nil
}

// GetStorageCurve 获取库容曲线
func (r *mapDataRepo) GetStorageCurve(ctx context.Context, reservoirID uint64) ([]*biz.CurvePoint, error) {
	var models []StorageCurvePoint
	if err := r.data.db.WithContext(ctx).Where("reservoir_id = ?", reservoirID).Order("level asc").Find(&models).Error; err != nil {
		r.log.Errorf("failed to get storage curve for reservoir ID %d: %v", reservoirID, err)
		return nil, err
	}

	bizPoints := make([]*biz.CurvePoint, len(models))
	for i, m := range models {
		// 传递数据库记录的 ID 到业务层
		idCopy := m.ID
		bizPoints[i] = &biz.CurvePoint{
			ID:    &idCopy, // 传递指针
			Level: m.Level,
			Value: m.Volume,
		}
	}
	return bizPoints, nil
}

// GetDischargeCurve 获取泄流曲线
func (r *mapDataRepo) GetDischargeCurve(ctx context.Context, reservoirID uint64) ([]*biz.CurvePoint, error) {
	var models []DischargeCurvePoint
	if err := r.data.db.WithContext(ctx).Where("reservoir_id = ?", reservoirID).Order("level asc").Find(&models).Error; err != nil {
		r.log.Errorf("failed to get discharge curve for reservoir ID %d: %v", reservoirID, err)
		return nil, err
	}

	bizPoints := make([]*biz.CurvePoint, len(models))
	for i, m := range models {
		idCopy := m.ID
		bizPoints[i] = &biz.CurvePoint{
			ID:    &idCopy,
			Level: m.Level,
			Value: m.Discharge,
		}
	}
	return bizPoints, nil
}

// UpdateCurves 更新曲线数据
func (r *mapDataRepo) UpdateCurves(ctx context.Context, reservoirID uint64, storageCurve []*biz.CurvePoint, dischargeCurve []*biz.CurvePoint) error {
	// 在事务中执行更新、插入和删除
	err := r.data.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// --- 更新库容曲线 ---
		if err := r.updateCurvePoints(tx, reservoirID, storageCurve, StorageCurvePoint{}); err != nil {
			return fmt.Errorf("failed to update storage curve: %w", err)
		}
		// --- 更新泄流曲线 ---
		if err := r.updateCurvePoints(tx, reservoirID, dischargeCurve, DischargeCurvePoint{}); err != nil {
			return fmt.Errorf("failed to update discharge curve: %w", err)
		}
		return nil // 事务成功
	})

	if err != nil {
		r.log.Errorf("UpdateCurves failed for reservoir %d: %v", reservoirID, err)
	}
	return err
}

// updateCurvePoints 是一个辅助函数，用于在事务中更新单个曲线
// modelType 用于确定目标表和字段映射 (通过空实例传递类型信息)
func (r *mapDataRepo) updateCurvePoints(tx *gorm.DB, reservoirID uint64, points []*biz.CurvePoint, modelType interface{}) error {
	idsToKeep := make(map[uint64]bool)
	var err error

	for _, p := range points {
		// 准备数据库模型
		var model interface{}
		var valueField string
		switch modelType.(type) {
		case StorageCurvePoint:
			model = &StorageCurvePoint{
				ReservoirID: reservoirID,
				Level:       p.Level,
				Volume:      p.Value, // 注意字段映射
			}
			valueField = "Volume"
		case DischargeCurvePoint:
			model = &DischargeCurvePoint{
				ReservoirID: reservoirID,
				Level:       p.Level,
				Discharge:   p.Value, // 注意字段映射
			}
			valueField = "Discharge"
		default:
			return fmt.Errorf("unsupported curve model type")
		}

		if p.ID != nil && *p.ID > 0 { // 如果提供了 ID，尝试更新
			currentID := *p.ID
			idsToKeep[currentID] = true

			// 根据类型设置主键和更新字段
			var result *gorm.DB
			switch m := model.(type) {
			case *StorageCurvePoint:
				m.ID = currentID
				result = tx.Model(m).Select("Level", valueField).Updates(m)
			case *DischargeCurvePoint:
				m.ID = currentID
				result = tx.Model(m).Select("Level", valueField).Updates(m)
			default:
				return fmt.Errorf("internal error: unexpected model type in update")
			}

			err = result.Error
			if err != nil {
				return fmt.Errorf("failed to update curve point with ID %d: %w", currentID, err)
			}
			if result.RowsAffected == 0 {
				// 如果没有行被更新，可能意味着记录不存在，尝试创建它
				r.log.Warnf("Curve point with ID %d not found for update, attempting to create.", currentID)
				if err = tx.Create(model).Error; err != nil {
					return fmt.Errorf("failed to create curve point after update attempt failed for ID %d: %w", currentID, err)
				}
			}
		} else { // 没有提供 ID，视为新增
			if err = tx.Create(model).Error; err != nil {
				return fmt.Errorf("failed to create new curve point: %w", err)
			}
		}
	}

	// 删除前端未提供的（即不在 idsToKeep 中的）旧记录
	var deleteResult *gorm.DB
	switch modelType.(type) {
	case StorageCurvePoint:
		deleteCond := tx.Where("reservoir_id = ?", reservoirID)
		if len(idsToKeep) > 0 {
			deleteCond = deleteCond.Where("id NOT IN ?", keysFromMap(idsToKeep))
		}
		deleteResult = deleteCond.Delete(&StorageCurvePoint{})
	case DischargeCurvePoint:
		deleteCond := tx.Where("reservoir_id = ?", reservoirID)
		if len(idsToKeep) > 0 {
			deleteCond = deleteCond.Where("id NOT IN ?", keysFromMap(idsToKeep))
		}
		deleteResult = deleteCond.Delete(&DischargeCurvePoint{})
	default:
		return fmt.Errorf("unsupported curve model type for delete")
	}

	if err = deleteResult.Error; err != nil {
		return fmt.Errorf("failed to delete old curve points: %w", err)
	}
	r.log.Infof("Deleted %d old curve points for reservoir %d", deleteResult.RowsAffected, reservoirID)

	return nil
}

// keysFromMap 是一个辅助函数，用于从 map[uint64]bool 获取所有 key
func keysFromMap(m map[uint64]bool) []uint64 {
	keys := make([]uint64, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// dereferenceFloat64 辅助函数 (如果 mapdata.go 中没有，需要添加)
// func dereferenceFloat64(f *float64) float64 { ... }

// --- 结束新增 ---
