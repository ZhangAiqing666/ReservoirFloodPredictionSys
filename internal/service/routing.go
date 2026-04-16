package service

import (
	// inflowV1 "ReservoirFloodPrediction/api/inflow/v1" // Not directly needed if request uses routingV1 types that embed inflow types
	routingV1 "ReservoirFloodPrediction/api/routing/v1" // Routing API
	"ReservoirFloodPrediction/internal/biz"             // Business logic
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// RoutingService 结构体实现了 routingV1.RoutingServiceServer 接口
type RoutingService struct {
	routingV1.UnimplementedRoutingServiceServer // 嵌入未实现的 server

	uc *biz.RoutingUseCase // 注入 RoutingUseCase
}

// NewRoutingService 创建一个新的 RoutingService
func NewRoutingService(uc *biz.RoutingUseCase) *RoutingService {
	return &RoutingService{uc: uc}
}

// RouteFlood 处理调洪计算请求
func (s *RoutingService) RouteFlood(ctx context.Context, req *routingV1.FloodRoutingRequest) (*routingV1.FloodRoutingReply, error) {
	// 1. 参数校验 (基础校验)
	if req == nil || req.ReservoirId == "" || len(req.InflowHydrograph) < 2 || req.InitialWaterLevel <= 0 {
		// TODO: Return specific gRPC error codes (e.g., codes.InvalidArgument)
		return &routingV1.FloodRoutingReply{Message: "invalid request parameters"}, fmt.Errorf("invalid request parameters")
	}

	// 2. 调用 UseCase 执行业务逻辑
	// 假设 biz.PerformFloodRouting 接收 reservoirID, inflow hydrograph, initial level
	// *** IMPORTANT: This assumes the signature of biz.PerformFloodRouting is ***
	// func (uc *RoutingUseCase) PerformFloodRouting(ctx context.Context, reservoirID string, inflowHydrograph []*inflowV1.HydrographDataPoint, initialWaterLevel float64) (*RoutingResult, error)
	// If not, the biz layer needs adjustment.

	// --- We need to modify biz.PerformFloodRouting signature ---
	// For now, let's proceed assuming it's modified or we'll modify it next.

	// ---> Placeholder call assuming modified signature <---
	// result, err := s.uc.PerformFloodRouting(ctx, req.ReservoirId, req.InflowHydrograph, req.InitialWaterLevel)

	// ===> Alternative: Get Params in Service, pass *biz.Reservoir to UseCase ===
	// This requires Service to depend on Repo, or UseCase to have a GetParams method.
	// Let's assume UseCase has GetParams for better encapsulation.
	// *** This requires adding GetReservoirParams to RoutingUseCase ***

	// ---> Placeholder assuming uc has GetReservoirParams method <---
	// reservoirParams, err := s.uc.GetReservoirParams(ctx, req.ReservoirId)
	// if err != nil {
	//     return &routingV1.FloodRoutingReply{Message: fmt.Sprintf("failed to get reservoir params: %v", err)}, err
	// }
	// result, err := s.uc.PerformFloodRouting(ctx, reservoirParams, req.InflowHydrograph, req.InitialWaterLevel)

	// ========> Simplest path for now: Modify PerformFloodRouting in biz <=========
	// Let's commit to modifying PerformFloodRouting in biz to accept reservoirID.
	// Assume the signature IS changed for the code below.

	result, err := s.uc.PerformFloodRouting(ctx, req.ReservoirId, req.InflowHydrograph, req.InitialWaterLevel)
	if err != nil {
		// TODO: Map biz errors to gRPC errors (e.g., NotFound, Internal)
		fmt.Printf("Error during flood routing calculation: %v\n", err) // Log the error
		return &routingV1.FloodRoutingReply{Message: fmt.Sprintf("routing calculation failed: %v", err)}, fmt.Errorf("routing calculation failed: %w", err)
	}

	// 3. 转换结果为 API 格式
	apiResults := make([]*routingV1.RoutingResultPoint, len(result.Results))
	for i, p := range result.Results {
		// Ensure Time is not zero before converting
		var ts *timestamppb.Timestamp
		if !p.Time.IsZero() {
			ts = timestamppb.New(p.Time)
		}
		apiResults[i] = &routingV1.RoutingResultPoint{
			Time:          ts,
			WaterLevel:    p.WaterLevel,
			StorageVolume: p.StorageVolume,
			Outflow:       p.Outflow,
			Inflow:        p.Inflow, // Include inflow if available in biz result
		}
	}

	// Ensure peak times are not zero before converting
	var peakLevelTs *timestamppb.Timestamp
	if !result.PeakWaterLevelTime.IsZero() {
		peakLevelTs = timestamppb.New(result.PeakWaterLevelTime)
	}
	var peakOutflowTs *timestamppb.Timestamp
	if !result.PeakOutflowTime.IsZero() {
		peakOutflowTs = timestamppb.New(result.PeakOutflowTime)
	}

	// 4. 构建并返回响应
	reply := &routingV1.FloodRoutingReply{
		Results:            apiResults,
		PeakWaterLevel:     result.PeakWaterLevel,
		PeakWaterLevelTime: peakLevelTs,
		PeakOutflow:        result.PeakOutflow,
		PeakOutflowTime:    peakOutflowTs,
		MaxStorageVolume:   result.MaxStorageVolume,
		Message:            "Flood routing calculation successful",
	}

	return reply, nil
}
