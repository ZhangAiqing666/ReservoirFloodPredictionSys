package biz

import (
	// mapdatav1 "ReservoirFloodPrediction/api/mapdata/v1" // Biz 层通常不直接依赖 API proto
	"context"
	// "time"
)

// --- 数据结构定义 ---

// BasinInfo 业务层使用的流域信息结构
type BasinInfo struct {
	ID              uint64 // 与数据库模型匹配
	BasinName       string
	ControlArea     float64
	MainStreamSlope float64
}

// ReservoirInfo 业务层使用的水库/雨量站信息结构
type ReservoirInfo struct {
	ID          uint64 // 与数据库模型匹配
	Name        string
	Latitude    float64
	Longitude   float64
	ControlArea float64 // 作为雨量站
	Weight      float64 // 作为雨量站
}

// CurvePoint 业务层使用的曲线点结构 (与 API 层一致)
type CurvePoint struct {
	ID    *uint64 // 使用指针，允许空表示新增
	Level float64
	Value float64
}

// --- Repository 接口定义 ---

// MapDataRepo 定义了 MapData 相关的数据访问接口
type MapDataRepo interface {
	ListBasins(ctx context.Context) ([]*BasinInfo, error)
	ListReservoirs(ctx context.Context) ([]*ReservoirInfo, error)

	// --- 新增接口方法 ---
	GenerateAndSaveCurves(ctx context.Context, reservoirID uint64) error
	GetStorageCurve(ctx context.Context, reservoirID uint64) ([]*CurvePoint, error)
	GetDischargeCurve(ctx context.Context, reservoirID uint64) ([]*CurvePoint, error)
	UpdateCurves(ctx context.Context, reservoirID uint64, storageCurve []*CurvePoint, dischargeCurve []*CurvePoint) error
	// --- 结束新增 ---
}

// --- UseCase 定义 --- (仅保留一个定义)

// MapDataUseCase 封装 MapData 相关业务逻辑
type MapDataUseCase struct {
	repo MapDataRepo // 依赖 Repository 接口
	// 可以注入 logger 等
}

// NewMapDataUseCase 创建 MapDataUseCase (仅保留一个定义)
func NewMapDataUseCase(repo MapDataRepo) *MapDataUseCase {
	return &MapDataUseCase{repo: repo}
}

// --- UseCase 方法实现 --- (仅保留一个定义)

func (uc *MapDataUseCase) ListBasins(ctx context.Context) ([]*BasinInfo, error) {
	return uc.repo.ListBasins(ctx)
}

func (uc *MapDataUseCase) ListReservoirs(ctx context.Context) ([]*ReservoirInfo, error) {
	return uc.repo.ListReservoirs(ctx)
}

// --- 新增 UseCase 方法实现 ---

// GenerateCurves 调用 Repo 生成并保存曲线
func (uc *MapDataUseCase) GenerateCurves(ctx context.Context, reservoirID uint64) error {
	// 这里可以添加额外的业务逻辑，例如检查水库是否存在等
	// 但目前直接调用 Repo 层
	return uc.repo.GenerateAndSaveCurves(ctx, reservoirID)
}

// GetCurves 获取指定水库的曲线
func (uc *MapDataUseCase) GetCurves(ctx context.Context, reservoirID uint64) (storageCurve []*CurvePoint, dischargeCurve []*CurvePoint, err error) {
	storageCurve, err = uc.repo.GetStorageCurve(ctx, reservoirID)
	if err != nil {
		return nil, nil, err // 如果获取库容曲线失败，直接返回错误
	}
	dischargeCurve, err = uc.repo.GetDischargeCurve(ctx, reservoirID)
	if err != nil {
		return nil, nil, err // 如果获取泄流曲线失败，直接返回错误
	}
	return storageCurve, dischargeCurve, nil
}

// UpdateCurves 更新指定水库的曲线
func (uc *MapDataUseCase) UpdateCurves(ctx context.Context, reservoirID uint64, storageCurve []*CurvePoint, dischargeCurve []*CurvePoint) error {
	// 这里可以添加验证逻辑，例如检查点数、单调性等
	return uc.repo.UpdateCurves(ctx, reservoirID, storageCurve, dischargeCurve)
}

// --- 结束新增 ---
