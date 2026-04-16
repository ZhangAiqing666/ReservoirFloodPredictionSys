package service

import (
	mapdatav1 "ReservoirFloodPrediction/api/mapdata/v1"
	"ReservoirFloodPrediction/internal/biz"
	"context"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/go-kratos/kratos/v2/log"
)

// MapDataService 结构体
type MapDataService struct {
	mapdatav1.UnimplementedMapDataServer                     // 嵌入未实现的 server
	uc                                   *biz.MapDataUseCase // 注入 UseCase
	log                                  *log.Helper
}

// NewMapDataService 创建 MapDataService
func NewMapDataService(uc *biz.MapDataUseCase, logger log.Logger) *MapDataService {
	return &MapDataService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/mapdata")),
	}
}

// ListBasins 处理获取流域列表的请求
func (s *MapDataService) ListBasins(ctx context.Context, req *mapdatav1.ListBasinsRequest) (*mapdatav1.ListBasinsReply, error) {
	bizBasins, err := s.uc.ListBasins(ctx)
	if err != nil {
		s.log.Errorf("ListBasins failed: %v", err)
		// TODO: Map error to gRPC status code
		return nil, err
	}

	apiBasins := make([]*mapdatav1.BasinInfo, len(bizBasins))
	for i, b := range bizBasins {
		apiBasins[i] = &mapdatav1.BasinInfo{
			Id:              b.ID,
			BasinName:       b.BasinName,
			ControlArea:     b.ControlArea,
			MainStreamSlope: b.MainStreamSlope,
		}
	}

	return &mapdatav1.ListBasinsReply{Basins: apiBasins}, nil
}

// ListReservoirs 处理获取水库/雨量站列表的请求
func (s *MapDataService) ListReservoirs(ctx context.Context, req *mapdatav1.ListReservoirsRequest) (*mapdatav1.ListReservoirsReply, error) {
	bizReservoirs, err := s.uc.ListReservoirs(ctx)
	if err != nil {
		s.log.Errorf("ListReservoirs failed: %v", err)
		// TODO: Map error to gRPC status code
		return nil, err
	}

	apiReservoirs := make([]*mapdatav1.ReservoirInfo, len(bizReservoirs))
	for i, r := range bizReservoirs {
		apiReservoirs[i] = &mapdatav1.ReservoirInfo{
			Id:          r.ID,
			Name:        r.Name,
			Latitude:    r.Latitude,
			Longitude:   r.Longitude,
			ControlArea: r.ControlArea,
			Weight:      r.Weight,
		}
	}

	return &mapdatav1.ListReservoirsReply{Reservoirs: apiReservoirs}, nil
}

// --- 新增 Service 方法实现 ---

// GenerateCurves 处理生成模拟曲线的请求
func (s *MapDataService) GenerateCurves(ctx context.Context, req *mapdatav1.GenerateCurvesRequest) (*mapdatav1.GenerateCurvesReply, error) {
	resID, err := strconv.ParseUint(req.ReservoirId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid reservoir ID format '%s': %w", req.ReservoirId, err)
	}

	err = s.uc.GenerateCurves(ctx, resID)
	if err != nil {
		s.log.Errorf("GenerateCurves failed for reservoir %s: %v", req.ReservoirId, err)
		// TODO: Map error to gRPC status code
		return &mapdatav1.GenerateCurvesReply{Message: fmt.Sprintf("Failed to generate curves: %v", err)}, err // 返回错误信息给前端
	}

	return &mapdatav1.GenerateCurvesReply{Message: "Simulated curves generated successfully for reservoir " + req.ReservoirId}, nil
}

// GetCurves 处理获取曲线数据的请求
func (s *MapDataService) GetCurves(ctx context.Context, req *mapdatav1.GetCurvesRequest) (*mapdatav1.GetCurvesReply, error) {
	resID, err := strconv.ParseUint(req.ReservoirId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid reservoir ID format '%s': %w", req.ReservoirId, err)
	}

	storageCurveBiz, dischargeCurveBiz, err := s.uc.GetCurves(ctx, resID)
	if err != nil {
		s.log.Errorf("GetCurves failed for reservoir %s: %v", req.ReservoirId, err)
		// TODO: Map error to gRPC status code
		return nil, err
	}

	// 转换 Biz 层 CurvePoint 为 API 层 CurvePoint
	storageCurveApi := make([]*mapdatav1.CurvePoint, len(storageCurveBiz))
	for i, p := range storageCurveBiz {
		storageCurveApi[i] = &mapdatav1.CurvePoint{
			Id:    wrapperspb.UInt64(dereferenceUint64Ptr(p.ID)), // 转换 ID
			Level: p.Level,
			Value: p.Value,
		}
	}

	dischargeCurveApi := make([]*mapdatav1.CurvePoint, len(dischargeCurveBiz))
	for i, p := range dischargeCurveBiz {
		dischargeCurveApi[i] = &mapdatav1.CurvePoint{
			Id:    wrapperspb.UInt64(dereferenceUint64Ptr(p.ID)), // 转换 ID
			Level: p.Level,
			Value: p.Value,
		}
	}

	return &mapdatav1.GetCurvesReply{
		StorageCurve:   storageCurveApi,
		DischargeCurve: dischargeCurveApi,
	}, nil
}

// UpdateCurves 处理更新曲线数据的请求
func (s *MapDataService) UpdateCurves(ctx context.Context, req *mapdatav1.UpdateCurvesRequest) (*mapdatav1.UpdateCurvesReply, error) {
	resID, err := strconv.ParseUint(req.ReservoirId, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid reservoir ID format '%s': %w", req.ReservoirId, err)
	}

	// 转换 API 层 CurvePoint 为 Biz 层 CurvePoint
	storageCurveBiz := make([]*biz.CurvePoint, len(req.StorageCurve))
	for i, p := range req.StorageCurve {
		storageCurveBiz[i] = &biz.CurvePoint{
			ID:    optionalUint64Ptr(p.Id), // 转换 ID
			Level: p.Level,
			Value: p.Value,
		}
	}

	dischargeCurveBiz := make([]*biz.CurvePoint, len(req.DischargeCurve))
	for i, p := range req.DischargeCurve {
		dischargeCurveBiz[i] = &biz.CurvePoint{
			ID:    optionalUint64Ptr(p.Id), // 转换 ID
			Level: p.Level,
			Value: p.Value,
		}
	}

	err = s.uc.UpdateCurves(ctx, resID, storageCurveBiz, dischargeCurveBiz)
	if err != nil {
		s.log.Errorf("UpdateCurves failed for reservoir %s: %v", req.ReservoirId, err)
		// TODO: Map error to gRPC status code
		return &mapdatav1.UpdateCurvesReply{Message: fmt.Sprintf("Failed to update curves: %v", err)}, err // 返回错误信息给前端
	}

	return &mapdatav1.UpdateCurvesReply{Message: "Curves updated successfully for reservoir " + req.ReservoirId}, nil
}

// --- 辅助函数 ---

// dereferenceUint64Ptr 安全地解引用 *uint64, nil 时返回 0
func dereferenceUint64Ptr(ptr *uint64) uint64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}

// optionalUint64Ptr 将 google.protobuf.UInt64Value 转换为 *uint64
// 如果输入为 nil 或值为 0 (通常表示未设置)，返回 nil 指针
func optionalUint64Ptr(val *wrapperspb.UInt64Value) *uint64 {
	if val == nil || val.Value == 0 {
		return nil
	}
	v := val.Value
	return &v
}

// --- 结束新增 ---
