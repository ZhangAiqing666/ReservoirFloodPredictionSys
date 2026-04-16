package service

import (
	"ReservoirFloodPrediction/internal/biz"
	"context"
	"errors"
	"strconv"
	"time"

	v1 "ReservoirFloodPrediction/api/inflow/v1" // 导入生成的 inflow v1 API 包

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InflowService 实现了 inflow.InflowServiceServer 接口
type InflowService struct {
	v1.UnimplementedInflowServiceServer // 必须嵌入

	uc  *biz.InflowUsecase // 依赖 InflowUsecase
	log *log.Helper
}

// NewInflowService 创建 InflowService 实例
func NewInflowService(uc *biz.InflowUsecase, logger log.Logger) *InflowService {
	return &InflowService{
		uc:  uc,
		log: log.NewHelper(log.With(logger, "module", "service/inflow")),
	}
}

// SimulateRainfall 实现模拟降雨接口
func (s *InflowService) SimulateRainfall(ctx context.Context, req *v1.SimulateRainfallRequest) (*v1.SimulateRainfallReply, error) {
	s.log.WithContext(ctx).Infof("Handling SimulateRainfall request, pattern: %s", req.RainPatternType)

	past24hBiz, past15dBiz, err := s.uc.SimulateRainfall(ctx, req.RainPatternType)
	if err != nil {
		s.log.WithContext(ctx).Errorf("SimulateRainfall failed: %v", err)
		return nil, err
	}

	past24hApi := convertBizToApiTimeSeries(past24hBiz)
	past15dApi := convertBizToApiTimeSeries(past15dBiz)

	return &v1.SimulateRainfallReply{
		Past_24HRainfall: past24hApi,
		Past_15DRainfall: past15dApi,
		Message:          "模拟降雨数据生成成功",
	}, nil
}

// PredictRainfall 实现预测降雨接口
func (s *InflowService) PredictRainfall(ctx context.Context, req *v1.PredictRainfallRequest) (*v1.PredictRainfallReply, error) {
	s.log.WithContext(ctx).Info("Handling PredictRainfall request")

	past24hBiz := convertApiToBizTimeSeries(req.Past_24HRainfall)

	next72hBiz, err := s.uc.PredictRainfall(ctx, past24hBiz)
	if err != nil {
		s.log.WithContext(ctx).Errorf("PredictRainfall failed: %v", err)
		return nil, err
	}

	next72hApi := convertBizToApiTimeSeries(next72hBiz)

	return &v1.PredictRainfallReply{
		Next_72HRainfall: next72hApi,
		Message:          "预测降雨数据生成成功",
	}, nil
}

// CalculateInflowVolume 实现计算入库洪量接口
func (s *InflowService) CalculateInflowVolume(ctx context.Context, req *v1.CalculateInflowVolumeRequest) (*v1.CalculateInflowVolumeReply, error) {
	s.log.WithContext(ctx).Infof("Handling CalculateInflowVolume request for basin ID: %s", req.BasinId)

	basinID, err := strconv.ParseUint(req.BasinId, 10, 64)
	if err != nil {
		s.log.WithContext(ctx).Errorf("Invalid basin ID format '%s': %v", req.BasinId, err)
		return nil, status.Errorf(codes.InvalidArgument, "invalid basin ID format: %s", req.BasinId)
	}

	rainfallDataBiz := convertApiToBizTimeSeries(req.RainfallData)

	// --- 调用业务层方法，接收洪量、过程线和洪峰信息 ---
	totalVolume, hydrographBiz, peakFlowBiz, peakFlowTimeBiz, err := s.uc.CalculateInflowVolume(ctx, basinID, rainfallDataBiz)
	if err != nil {
		s.log.WithContext(ctx).Errorf("CalculateInflowVolume usecase failed: %v", err)
		if errors.Is(err, biz.ErrBasinInfoNotFound) {
			return nil, status.Errorf(codes.NotFound, "basin info not found for ID: %d", basinID)
		}
		return nil, status.Errorf(codes.Internal, "failed to calculate inflow volume: %v", err)
	}
	// -----------------------------------------------

	hydrographApi := convertBizToApiHydrograph(hydrographBiz)

	// --- 转换洪峰时间 ---
	var peakFlowTimeApi *timestamppb.Timestamp
	if !peakFlowTimeBiz.IsZero() { // 仅当时间非零值时才转换
		peakFlowTimeApi = timestamppb.New(peakFlowTimeBiz)
	}
	// -------------------

	return &v1.CalculateInflowVolumeReply{
		TotalInflowVolume: totalVolume,
		HydrographData:    hydrographApi,
		PeakFlow:          peakFlowBiz,        // <--- 添加洪峰流量
		PeakFlowTime:      peakFlowTimeApi,    // <--- 添加洪峰时间
		Message:           "入库总洪量、过程线及洪峰计算成功", // 更新消息
	}, nil
}

// --- 辅助函数：用于 biz 和 api 结构体之间的转换 ---

func convertBizToApiTimeSeries(bizData []*biz.TimeSeriesDataPoint) []*v1.TimeSeriesDataPoint {
	apiData := make([]*v1.TimeSeriesDataPoint, 0, len(bizData))
	for _, p := range bizData {
		if p == nil {
			continue
		}
		apiData = append(apiData, &v1.TimeSeriesDataPoint{
			Time:  timestamppb.New(p.Time),
			Value: p.Value,
		})
	}
	return apiData
}

func convertBizToApiHydrograph(bizData []*biz.HydrographPoint) []*v1.HydrographDataPoint {
	apiData := make([]*v1.HydrographDataPoint, 0, len(bizData))
	for _, p := range bizData {
		if p == nil {
			continue
		}
		apiData = append(apiData, &v1.HydrographDataPoint{
			Time:     timestamppb.New(p.Time),
			FlowRate: p.FlowRate,
		})
	}
	return apiData
}

func convertApiToBizTimeSeries(apiData []*v1.TimeSeriesDataPoint) []*biz.TimeSeriesDataPoint {
	bizData := make([]*biz.TimeSeriesDataPoint, 0, len(apiData))
	for _, p := range apiData {
		if p == nil {
			continue
		}
		// 检查 Time 字段是否有效
		var t time.Time
		if p.Time != nil && p.Time.IsValid() {
			t = p.Time.AsTime()
		} else {
			// 如果时间无效，可以记录警告或跳过该点
			// log.Warnf("Invalid timestamp received in API data: %v", p.Time)
			continue // 或者使用一个默认时间？当前选择跳过
		}
		bizData = append(bizData, &biz.TimeSeriesDataPoint{
			Time:  t,
			Value: p.Value,
		})
	}
	return bizData
}
