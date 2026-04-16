package biz

import (
	"ReservoirFloodPrediction/internal/conf"
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	// "google.golang.org/protobuf/types/known/timestamppb" // biz 层不需要直接用 protobuf 类型
)

// TimeSeriesDataPoint 定义业务层的时间序列数据点结构
type TimeSeriesDataPoint struct {
	Time  time.Time
	Value float64
}

// HydrographPoint 定义业务层的水文过程线数据点结构
type HydrographPoint struct {
	Time     time.Time
	FlowRate float64 // 流量，单位: m³/s
}

// InflowUsecase 包含入库洪水分析相关的业务逻辑
type InflowUsecase struct {
	mapRepo MapDataRepo // 依赖 MapDataRepo 获取流域和水库信息
	cfg     *conf.Biz   // 可以从配置文件读取参数，例如产流系数
	log     *log.Helper
}

// NewInflowUsecase 创建 InflowUsecase 实例
func NewInflowUsecase(repo MapDataRepo, cfg *conf.Biz, logger log.Logger) *InflowUsecase {
	// --- 添加随机数种子初始化 ---
	// 使用当前时间作为种子，确保每次程序启动时随机序列不同
	// 注意：这在 Go 1.20+ 不是必需的，因为全局 rand 使用了自动种子。
	// 但为了兼容性和明确性，可以保留。
	// 如果使用的是 Go 1.20+ 并且不希望依赖全局 rand，可以使用 rand.New(rand.NewSource(time.Now().UnixNano()))
	rand.Seed(time.Now().UnixNano())
	// --------------------------

	if cfg == nil {
		log.NewHelper(logger).Warn("Biz config is nil, using default runoff coefficient.")
		cfg = &conf.Biz{
			DefaultRunoffCoefficient: 0.6,
		}
	}
	return &InflowUsecase{
		mapRepo: repo,
		cfg:     cfg,
		log:     log.NewHelper(log.With(logger, "module", "usecase/inflow")),
	}
}

// SimulateRainfall 根据雨型模拟历史降雨
func (uc *InflowUsecase) SimulateRainfall(ctx context.Context, rainPatternType string) (past24h []*TimeSeriesDataPoint, past15d []*TimeSeriesDataPoint, err error) {
	uc.log.WithContext(ctx).Infof("Simulating rainfall for pattern: %s", rainPatternType)

	now := time.Now()
	// 使用预分配容量初始化切片
	past24h = make([]*TimeSeriesDataPoint, 0, 24)
	past15d = make([]*TimeSeriesDataPoint, 0, 15)

	// --- 模拟过去 24 小时 (按小时) ---
	uc.log.WithContext(ctx).Debugf("Simulating past 24 hours from: %v", now)
	for i := 0; i < 24; i++ {
		// 时间从 1 小时前到 24 小时前
		t := now.Add(-time.Duration(i+1) * time.Hour)
		var rainfall float64
		switch rainPatternType {
		case "heavy_burst":
			// 假设最近 6 小时是暴雨高峰
			if i < 6 {
				rainfall = 10 + rand.Float64()*15 // 10 到 25 mm
			} else {
				rainfall = rand.Float64() * 2 // 0 到 2 mm
			}
		case "moderate_prolonged":
			rainfall = 2 + rand.Float64()*4 // 2 到 6 mm
		case "light_drizzle":
			rainfall = rand.Float64() * 1.5 // 0 到 1.5 mm
		default:
			uc.log.WithContext(ctx).Warnf("Unknown rain pattern type '%s', using default simulation.", rainPatternType)
			rainfall = rand.Float64() * 1 // 0 到 1 mm
		}
		// 确保雨量不为负
		if rainfall < 0 {
			rainfall = 0
		}
		// --- 添加日志 ---
		uc.log.WithContext(ctx).Debugf("Generated past 24h point: Time=%v, Value=%.2f", t, rainfall)
		// -------------
		past24h = append(past24h, &TimeSeriesDataPoint{Time: t, Value: rainfall})
	}
	// 反转切片，使时间从最早到最近
	reverseTimeSeries(past24h)
	uc.log.WithContext(ctx).Infof("Generated %d points for past 24h rainfall.", len(past24h))

	// --- 模拟过去 15 天 (按天) ---
	uc.log.WithContext(ctx).Debugf("Simulating past 15 days from: %v", now)
	for i := 0; i < 15; i++ {
		// 时间从 1 天前到 15 天前
		t := now.AddDate(0, 0, -(i + 1))
		// 将时间设为当天的 0 点
		dayStart := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		var dailyRainfall float64
		switch rainPatternType {
		case "heavy_burst":
			// 假设最近 2 天雨最大
			if i < 2 {
				dailyRainfall = 50 + rand.Float64()*50 // 50 到 100 mm
			} else if i < 5 { // 接下来 3 天雨量减少
				dailyRainfall = 10 + rand.Float64()*20 // 10 到 30 mm
			} else { // 更早的时间雨量很小
				dailyRainfall = rand.Float64() * 5 // 0 到 5 mm
			}
		case "moderate_prolonged":
			// 假设持续 10 天中雨
			if i < 10 {
				dailyRainfall = 15 + rand.Float64()*15 // 15 到 30 mm
			} else {
				dailyRainfall = rand.Float64() * 5 // 0 到 5 mm
			}
		case "light_drizzle":
			dailyRainfall = 1 + rand.Float64()*4 // 1 到 5 mm
		default:
			dailyRainfall = rand.Float64() * 3 // 0 到 3 mm
		}
		// 确保雨量不为负
		if dailyRainfall < 0 {
			dailyRainfall = 0
		}
		// --- 添加日志 ---
		uc.log.WithContext(ctx).Debugf("Generated past 15d point: Time=%v, Value=%.2f", dayStart, dailyRainfall)
		// -------------
		past15d = append(past15d, &TimeSeriesDataPoint{Time: dayStart, Value: dailyRainfall})
	}
	// 反转切片，使时间从最早到最近
	reverseTimeSeries(past15d)
	uc.log.WithContext(ctx).Infof("Generated %d points for past 15d rainfall.", len(past15d))

	// 确认返回值
	if len(past24h) == 0 || len(past15d) == 0 {
		uc.log.WithContext(ctx).Warnf("SimulateRainfall is returning potentially empty slices: len(past24h)=%d, len(past15d)=%d", len(past24h), len(past15d))
	}

	return past24h, past15d, nil
}

// PredictRainfall 预测未来降雨 (简化逻辑)
func (uc *InflowUsecase) PredictRainfall(ctx context.Context, past24hData []*TimeSeriesDataPoint) (next72h []*TimeSeriesDataPoint, err error) {
	uc.log.WithContext(ctx).Info("Predicting future rainfall based on simulated past data")
	next72h = make([]*TimeSeriesDataPoint, 0, 72)

	if len(past24hData) == 0 {
		uc.log.WithContext(ctx).Warn("No past 24h data provided for prediction, returning empty prediction.")
		return next72h, nil // 如果没有历史数据，无法预测
	}

	// 简化的预测逻辑：基于过去6小时平均值进行衰减预测
	last6hSum := 0.0
	count := 0
	// 计算过去 6 个数据点（如果不足6个，则计算所有）
	startIndex := len(past24hData) - 6
	if startIndex < 0 {
		startIndex = 0
	}
	for i := startIndex; i < len(past24hData); i++ {
		last6hSum += past24hData[i].Value
		count++
	}
	avgLast6h := 0.0
	if count > 0 {
		avgLast6h = last6hSum / float64(count)
		uc.log.WithContext(ctx).Debugf("Average rainfall in last %d hours: %.2f", count, avgLast6h)
	} else {
		uc.log.WithContext(ctx).Warn("Could not calculate average rainfall from past data.")
		// 即使无法计算平均值，也可能需要生成一些小的随机值或返回错误
	}

	now := time.Now() // 预测从当前时间开始
	for i := 0; i < 72; i++ {
		// 时间从 1 小时后到 72 小时后
		t := now.Add(time.Duration(i+1) * time.Hour)
		var predictedRainfall float64
		// 简单的衰减模型 + 随机扰动
		decayFactor := 1.0
		if i < 24 { // 第一个 24 小时衰减较慢
			decayFactor = (0.8 - 0.4*float64(i)/24.0)
		} else if i < 48 { // 第二个 24 小时衰减加快
			decayFactor = (0.4 - 0.3*float64(i-24)/24.0)
		} else { // 第三个 24 小时衰减更快
			decayFactor = (0.1 - 0.1*float64(i-48)/24.0)
		}
		// 基本预测值 = 平均值 * 衰减因子
		predictedRainfall = avgLast6h * decayFactor
		// 添加小的随机扰动（平均值的 +/- 5%）
		predictedRainfall += rand.Float64()*avgLast6h*0.1 - avgLast6h*0.05

		if predictedRainfall < 0 {
			predictedRainfall = 0
		}
		// --- 添加日志 ---
		uc.log.WithContext(ctx).Debugf("Generated predicted point: Time=%v, Value=%.2f", t, predictedRainfall)
		// -------------
		next72h = append(next72h, &TimeSeriesDataPoint{Time: t, Value: predictedRainfall})
	}
	uc.log.WithContext(ctx).Infof("Generated %d points for next 72h prediction.", len(next72h))

	return next72h, nil
}

// GenerateInflowHydrograph 根据降雨数据和流域信息计算入库流量过程线 (简化模型)
func (uc *InflowUsecase) GenerateInflowHydrograph(ctx context.Context, rainfallData []*TimeSeriesDataPoint, basinControlArea float64, runoffCoefficient float64) ([]*HydrographPoint, error) {
	uc.log.WithContext(ctx).Infof("Generating inflow hydrograph with %d rainfall points, Area=%.2f, Coeff=%.2f", len(rainfallData), basinControlArea, runoffCoefficient)
	hydrograph := make([]*HydrographPoint, 0, len(rainfallData))

	if len(rainfallData) == 0 {
		uc.log.WithContext(ctx).Warn("No rainfall data provided for hydrograph generation.")
		return hydrograph, nil // 返回空过程线
	}

	// 简化模型参数 (可以根据实际情况调整或从配置读取)
	influenceHours := 3                 // 例如，当前小时流量受当前及过去2小时（共3小时）降雨影响
	weights := []float64{0.5, 0.3, 0.2} // 权重（越近的降雨影响越大），总和应为 1.0

	if len(weights) != influenceHours {
		uc.log.WithContext(ctx).Error("Hydrograph model parameter mismatch (influenceHours vs weights length).")
		return nil, errors.New("hydrograph model parameter mismatch")
	}

	// 准备有效降雨数据 (单位: mm/hour)
	effectiveRainfall := make([]float64, len(rainfallData))
	for i, point := range rainfallData {
		if point != nil { // 添加 nil 检查
			effectiveRainfall[i] = point.Value * runoffCoefficient // 假设 point.Value 是小时降雨量 mm
		} else {
			effectiveRainfall[i] = 0
		}
	}

	// 计算每个时间点的流量
	for i := 0; i < len(rainfallData); i++ {
		currentFlow := 0.0
		// 应用加权平均模拟汇流
		for j := 0; j < influenceHours; j++ {
			rainfallIndex := i - j
			if rainfallIndex >= 0 {
				// 流量 (m³/s) = 有效降雨量 (mm/hr) * 流域面积 (km²) / 3.6
				currentFlow += effectiveRainfall[rainfallIndex] * basinControlArea / 3.6 * weights[j]
			}
		}

		if currentFlow < 0 {
			currentFlow = 0
		}

		// 确保 rainfallData[i] 不是 nil
		if rainfallData[i] == nil {
			uc.log.WithContext(ctx).Warnf("Skipping hydrograph point generation at index %d due to nil rainfall data", i)
			continue
		}

		hydrographPoint := &HydrographPoint{
			Time:     rainfallData[i].Time,
			FlowRate: currentFlow,
		}
		hydrograph = append(hydrograph, hydrographPoint)
		uc.log.WithContext(ctx).Debugf("Generated hydrograph point: Time=%v, FlowRate=%.2f m³/s", hydrographPoint.Time, hydrographPoint.FlowRate)
	}

	uc.log.WithContext(ctx).Infof("Generated %d hydrograph points.", len(hydrograph))
	return hydrograph, nil
}

// CalculateInflowVolume 计算总入库洪量、过程线及洪峰信息
func (uc *InflowUsecase) CalculateInflowVolume(ctx context.Context, basinID uint64, rainfallData []*TimeSeriesDataPoint) (
	totalVolume float64,
	hydrographData []*HydrographPoint,
	peakFlow float64, // <-- 新增返回值：洪峰流量
	peakFlowTime time.Time, // <-- 新增返回值：洪峰时间
	err error,
) { // <--- 修改返回值签名
	uc.log.WithContext(ctx).Infof("Calculating inflow volume and hydrograph for basin ID: %d", basinID)

	if len(rainfallData) == 0 {
		uc.log.WithContext(ctx).Error("Rainfall data is required for calculation")
		return 0, nil, 0, time.Time{}, errors.New("rainfall data is required for calculation")
	}

	// !!! 临时修复编译错误：先注释掉调用，并使用默认值。后续需要完善业务逻辑 !!!
	// basinInfo, err := uc.mapRepo.ListBasins(ctx) // 编译会报错，因为 ListBasins 返回列表
	// if err != nil {
	// 	return 0, nil, 0, time.Time{}, fmt.Errorf("failed to get basin info for ID %d: %w", basinID, err)
	// }
	// // 需要从列表中找到对应的 basinID，或者修改 repo 接口
	// controlArea := 0.0 // 临时值
	// // ... 找到对应 basinID 的 controlArea ...

	// 假设直接传入控制面积（或从其他地方获取），暂时绕过 mapRepo 调用
	// !!! 注意：这里的控制面积需要从调用者或配置传入，或者完善 GetBasinByID 逻辑 !!!
	controlArea := 100.0 // <<--- ！！！ 这是一个占位符/默认值 ！！！
	uc.log.WithContext(ctx).Warnf("Using placeholder control area: %.2f for basin ID %d", controlArea, basinID)

	runoffCoefficient32 := uc.cfg.GetDefaultRunoffCoefficient()
	if runoffCoefficient32 <= 0 || runoffCoefficient32 > 1 {
		uc.log.WithContext(ctx).Warnf("Runoff coefficient (%.2f) from config is invalid, using default 0.6", runoffCoefficient32)
		runoffCoefficient32 = 0.6
	}
	runoffCoefficient := float64(runoffCoefficient32)
	uc.log.WithContext(ctx).Debugf("Using Runoff Coefficient: %.2f", runoffCoefficient)

	totalNetRainfallMM := 0.0
	for _, point := range rainfallData {
		if point != nil {
			totalNetRainfallMM += point.Value * runoffCoefficient
		}
	}
	uc.log.WithContext(ctx).Debugf("Total Net Rainfall: %.2f mm", totalNetRainfallMM)

	totalVolume = totalNetRainfallMM * controlArea / 10.0
	if totalVolume < 0 {
		uc.log.WithContext(ctx).Warnf("Calculated total volume is negative (%.2f), setting to 0.", totalVolume)
		totalVolume = 0
	}
	uc.log.WithContext(ctx).Infof("Calculated total inflow volume: %.2f 万立方米", totalVolume)

	hydrographData, hydrographErr := uc.GenerateInflowHydrograph(ctx, rainfallData, controlArea, runoffCoefficient)
	if hydrographErr != nil {
		uc.log.WithContext(ctx).Errorf("Failed to generate inflow hydrograph: %v", hydrographErr)
		err = hydrographErr
		// 即使过程线生成失败，也尝试返回洪量，但洪峰信息将是零值
		return totalVolume, hydrographData, 0, time.Time{}, err // <--- 确保在错误时返回零值洪峰
	}

	// --- 新增：查找洪峰流量和时间 ---
	peakFlow = 0.0
	if len(hydrographData) > 0 {
		peakFlow = hydrographData[0].FlowRate // 假设第一个点为初始洪峰
		peakFlowTime = hydrographData[0].Time
		for _, point := range hydrographData {
			if point != nil && point.FlowRate > peakFlow {
				peakFlow = point.FlowRate
				peakFlowTime = point.Time
			}
		}
		uc.log.WithContext(ctx).Infof("Peak flow found: %.2f m³/s at %v", peakFlow, peakFlowTime)
	} else {
		uc.log.WithContext(ctx).Warn("Cannot find peak flow from empty hydrograph data.")
		// peakFlow 保持 0, peakFlowTime 保持零值
	}
	// -----------------------------

	return totalVolume, hydrographData, peakFlow, peakFlowTime, err // <--- 修改返回值
}

// reverseTimeSeries 原地反转时间序列切片
func reverseTimeSeries(s []*TimeSeriesDataPoint) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}
