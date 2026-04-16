package biz

import (
	inflowV1 "ReservoirFloodPrediction/api/inflow/v1"   // 使用别名 inflowV1
	routingV1 "ReservoirFloodPrediction/api/routing/v1" // 使用别名 routingV1
	"context"
	"fmt"
	"math" // 需要 math 包
	"sort" // 需要 sort 包
	"time"
	// 需要 timestamp 包
)

// --- 数据结构 ---

// ReservoirParams 定义了获取水库参数的接口返回结构 (如果需要单独获取)
// 如果 RoutingRepo 直接返回 biz.Reservoir, 则此类型可能不需要
type ReservoirParams struct {
	ID                      string
	Name                    string
	StorageCurve            []*routingV1.CurvePoint         // 使用 routingV1 别名
	DischargeCurve          []*routingV1.CurvePoint         // 使用 routingV1 别名
	Levels                  *routingV1.CharacteristicLevels // 使用 routingV1 别名
	DownstreamSafeDischarge float64
}

// Reservoir 是业务逻辑层内部使用的水库结构 (导出)
// 使用大写字母开头使其可导出
type Reservoir struct {
	ID                      string
	Name                    string
	StorageCurve            []*routingV1.CurvePoint // 水位(level) vs 库容(value)
	DischargeCurve          []*routingV1.CurvePoint // 水位(level) vs 下泄流量(value)
	Levels                  *routingV1.CharacteristicLevels
	DownstreamSafeDischarge float64
}

// RoutingResult 保存调洪计算的完整结果 (业务逻辑层)
type RoutingResult struct {
	Results            []RoutingResultPoint // 过程线
	PeakWaterLevel     float64
	PeakWaterLevelTime time.Time
	PeakOutflow        float64
	PeakOutflowTime    time.Time
	MaxStorageVolume   float64
}

// RoutingResultPoint 调洪结果的一个时间点 (业务逻辑层)
type RoutingResultPoint struct {
	Time          time.Time
	WaterLevel    float64 // m
	StorageVolume float64 // 10^4 m³
	Outflow       float64 // m³/s
	Inflow        float64 // m³/s
}

// --- UseCase 定义 ---

// RoutingRepo 定义了数据访问接口 (已导出)
type RoutingRepo interface {
	// GetReservoirParams 返回 biz 层使用的水库结构
	GetReservoirParams(ctx context.Context, reservoirID string) (*Reservoir, error) // 返回导出的 Reservoir 类型
}

// RoutingUseCase 封装了调洪计算的业务逻辑 (已导出)
type RoutingUseCase struct {
	repo RoutingRepo // 数据仓库接口
	// 可以注入 logger 等其他依赖
}

// NewRoutingUseCase 创建一个新的 RoutingUseCase
func NewRoutingUseCase(repo RoutingRepo) *RoutingUseCase {
	return &RoutingUseCase{repo: repo}
}

// --- 核心计算逻辑 ---

// Helper function for linear interpolation
// x: the value to interpolate at (e.g., level)
// points: sorted slice of *routingV1.CurvePoint (sorted by x-value, i.e., level)
// returns the interpolated y-value (e.g., storage or discharge)
func interpolate(x float64, points []*routingV1.CurvePoint) float64 {
	if len(points) == 0 {
		return 0 // Or handle error
	}
	// Sort points just in case they are not (by level)
	sort.SliceStable(points, func(i, j int) bool {
		// Handle nil points defensively
		if points[i] == nil || points[j] == nil {
			return false // Or decide how to handle nil
		}
		return points[i].Level < points[j].Level
	})

	// Remove nil points after potential sorting issues or if they exist initially
	validPoints := make([]*routingV1.CurvePoint, 0, len(points))
	for _, p := range points {
		if p != nil {
			validPoints = append(validPoints, p)
		}
	}
	points = validPoints
	if len(points) == 0 {
		return 0
	}

	// Handle edge cases: below min or above max
	if x <= points[0].Level {
		return points[0].Value
	}
	if x >= points[len(points)-1].Level {
		return points[len(points)-1].Value
	}

	// Find the segment for interpolation
	for i := 0; i < len(points)-1; i++ {
		if x >= points[i].Level && x <= points[i+1].Level {
			x1, y1 := points[i].Level, points[i].Value
			x2, y2 := points[i+1].Level, points[i+1].Value
			if math.Abs(x2-x1) < 1e-9 { // Avoid division by zero if levels are too close
				return y1 // Or average: (y1+y2)/2
			}
			// Linear interpolation formula: y = y1 + (x - x1) * (y2 - y1) / (x2 - x1)
			return y1 + (x-x1)*(y2-y1)/(x2-x1)
		}
	}
	// Should not reach here if points are sorted and x is within bounds
	return points[len(points)-1].Value // Fallback
}

// getStorageByLevel calculates storage volume (10^4 m³) for a given water level (m)
func getStorageByLevel(level float64, storageCurve []*routingV1.CurvePoint) float64 {
	return interpolate(level, storageCurve)
}

// getLevelByStorage calculates water level (m) for a given storage volume (10^4 m³)
func getLevelByStorage(storage float64, storageCurve []*routingV1.CurvePoint) float64 {
	if len(storageCurve) == 0 {
		return 0 // Or handle error
	}
	// Sort points by storage value to interpolate level
	sort.SliceStable(storageCurve, func(i, j int) bool {
		if storageCurve[i] == nil || storageCurve[j] == nil {
			return false
		}
		return storageCurve[i].Value < storageCurve[j].Value // Sort by storage
	})

	// Filter nils again after sorting by value
	validPoints := make([]*routingV1.CurvePoint, 0, len(storageCurve))
	for _, p := range storageCurve {
		if p != nil {
			validPoints = append(validPoints, p)
		}
	}
	storageCurve = validPoints
	if len(storageCurve) == 0 {
		return 0
	}

	if storage <= storageCurve[0].Value {
		return storageCurve[0].Level
	}
	if storage >= storageCurve[len(storageCurve)-1].Value {
		return storageCurve[len(storageCurve)-1].Level
	}

	for i := 0; i < len(storageCurve)-1; i++ {
		if storage >= storageCurve[i].Value && storage <= storageCurve[i+1].Value {
			x1, y1 := storageCurve[i].Value, storageCurve[i].Level // x is storage, y is level
			x2, y2 := storageCurve[i+1].Value, storageCurve[i+1].Level
			if math.Abs(x2-x1) < 1e-9 { // Avoid division by zero if values are too close
				return y1
			}
			return y1 + (storage-x1)*(y2-y1)/(x2-x1)
		}
	}
	return storageCurve[len(storageCurve)-1].Level // Fallback
}

// getDischargeByLevel calculates outflow (m³/s) for a given water level (m)
// TODO: Implement actual scheduling rules here. For now, assume free discharge based on curve.
func getDischargeByLevel(level float64, dischargeCurve []*routingV1.CurvePoint, downstreamSafeDischarge float64, floodLimitWaterLevel float64) float64 {
	q := interpolate(level, dischargeCurve)

	// Basic Rule Example (can be refined):
	// if level <= floodLimitWaterLevel {
	//     // Below flood limit, perhaps minimal outflow or zero outflow
	//     // For simplicity, let's assume it still follows the curve for now,
	//     // but real rules would apply here (e.g., keep outflow = 0 or = min ecological flow)
	//     // q = 0 // Example: Keep closed below FSL
	// }

	// Optional: Limit outflow if downstream safe discharge is defined and exceeded.
	// Be careful with this, as it might cause the level to rise unrealistically if inflow is high.
	// A better approach involves complex gate operation rules.
	if downstreamSafeDischarge > 0 && q > downstreamSafeDischarge {
		// fmt.Printf("Warning: Discharge %.2f exceeds safe limit %.2f at level %.2f. Limiting outflow.\n", q, downstreamSafeDischarge, level)
		// q = downstreamSafeDischarge // Uncomment carefully
	}
	return q
}

// PerformFloodRouting performs flood routing using the level pool method (water balance)
// Now accepts reservoirID string and fetches params internally via repo.
func (uc *RoutingUseCase) PerformFloodRouting(ctx context.Context, reservoirID string, inflowHydrograph []*inflowV1.HydrographDataPoint, initialWaterLevel float64) (*RoutingResult, error) {
	// 1. Get Reservoir Params using the injected repo
	reservoir, err := uc.repo.GetReservoirParams(ctx, reservoirID)
	if err != nil {
		return nil, fmt.Errorf("failed to get reservoir params for ID %s: %w", reservoirID, err)
	}
	if reservoir == nil { // Double check after error check
		return nil, fmt.Errorf("reservoir parameters were nil after fetching for ID %s", reservoirID)
	}

	if len(inflowHydrograph) < 2 {
		return nil, fmt.Errorf("insufficient inflow hydrograph data (need at least 2 points)")
	}
	if len(reservoir.StorageCurve) < 2 || len(reservoir.DischargeCurve) < 2 {
		return nil, fmt.Errorf("insufficient reservoir curve data (need at least 2 points per curve)")
	}
	if reservoir.Levels == nil {
		// Provide default levels or return error if critical levels are missing
		fmt.Println("Warning: Reservoir characteristic levels are nil. Using defaults or 0.")
		reservoir.Levels = &routingV1.CharacteristicLevels{} // Assign empty struct to avoid nil pointer later
	}

	// 1. Data Preparation & Sort Inflow
	sort.SliceStable(inflowHydrograph, func(i, j int) bool {
		if inflowHydrograph[i] == nil || inflowHydrograph[j] == nil || inflowHydrograph[i].Time == nil || inflowHydrograph[j].Time == nil {
			return false
		}
		return inflowHydrograph[i].Time.AsTime().Before(inflowHydrograph[j].Time.AsTime())
	})

	// Filter nil entries in inflow
	validInflow := make([]*inflowV1.HydrographDataPoint, 0, len(inflowHydrograph))
	for _, p := range inflowHydrograph {
		if p != nil && p.Time != nil {
			validInflow = append(validInflow, p)
		}
	}
	inflowHydrograph = validInflow
	if len(inflowHydrograph) < 2 {
		return nil, fmt.Errorf("insufficient valid inflow hydrograph data after filtering")
	}

	results := make([]RoutingResultPoint, 0, len(inflowHydrograph))
	peakWaterLevel := initialWaterLevel
	peakWaterLevelTime := time.Time{} // Zero time initially
	peakOutflow := 0.0
	peakOutflowTime := time.Time{}
	maxStorageVolume := 0.0

	// 2. Initialization
	currentTime := inflowHydrograph[0].Time.AsTime()
	currentLevel := initialWaterLevel
	currentStorage := getStorageByLevel(currentLevel, reservoir.StorageCurve)
	currentOutflow := getDischargeByLevel(currentLevel, reservoir.DischargeCurve, reservoir.DownstreamSafeDischarge, reservoir.Levels.FloodLimitWaterLevel) // Initial outflow
	currentInflow := inflowHydrograph[0].FlowRate                                                                                                           // Inflow at t1

	if currentTime.IsZero() {
		return nil, fmt.Errorf("invalid start time in inflow hydrograph")
	}
	peakWaterLevelTime = currentTime // Initialize peak time
	peakOutflowTime = currentTime

	// Add initial point to results
	initialPoint := RoutingResultPoint{
		Time:          currentTime,
		WaterLevel:    currentLevel,
		StorageVolume: currentStorage,
		Outflow:       currentOutflow,
		Inflow:        currentInflow,
	}
	results = append(results, initialPoint)
	maxStorageVolume = currentStorage
	peakOutflow = currentOutflow

	// 3. Time Stepping Calculation using Water Balance Equation:
	// (I1 + I2)/2 * dt - (O1 + O2)/2 * dt = S2 - S1
	// Rearranged: S2 + O2/2 * dt = S1 - O1/2 * dt + (I1 + I2)/2 * dt
	// Let LHS(Z2) = S(Z2) + Q(Z2)/2 * dt
	// Let RHS = S1 - O1/2 * dt + (I1 + I2)/2 * dt (known)
	// Solve LHS(Z2) = RHS for Z2

	fmt.Printf("Starting routing. Initial Level: %.3f, Storage: %.3f, Outflow: %.3f\n", currentLevel, currentStorage, currentOutflow)

	for i := 0; i < len(inflowHydrograph)-1; i++ {
		// Ensure points and time are valid
		if inflowHydrograph[i+1] == nil || inflowHydrograph[i+1].Time == nil {
			fmt.Printf("Warning: Skipping step %d due to nil data point or time.\n", i+1)
			continue
		}
		t1 := inflowHydrograph[i].Time.AsTime()
		t2 := inflowHydrograph[i+1].Time.AsTime()
		if t1.IsZero() || t2.IsZero() {
			fmt.Printf("Warning: Skipping step %d due to zero time.\n", i+1)
			continue
		}
		dtSeconds := t2.Sub(t1).Seconds()
		if dtSeconds <= 1e-9 { // Use tolerance instead of direct zero check
			fmt.Printf("Warning: Skipping step %d due to non-positive time difference (%.3f seconds).\n", i+1, dtSeconds)
			continue // Skip if time step is not positive or too small
		}

		I1 := inflowHydrograph[i].FlowRate
		I2 := inflowHydrograph[i+1].FlowRate
		O1 := currentOutflow
		S1 := currentStorage
		Z1 := currentLevel

		avgInflow := (I1 + I2) / 2.0
		rhs := S1 - (O1/2.0)*dtSeconds + avgInflow*dtSeconds // Calculate the known right-hand side
		// Note: Storage is in 10^4 m³, Flow is in m³/s.
		// If dt is in seconds, the equation units must match.
		// S units: 10^4 m³ = 10000 m³
		// Q units: m³/s
		// dt units: s
		// (I1+I2)/2 * dt -> m³/s * s = m³
		// (O1+O2)/2 * dt -> m³/s * s = m³
		// S2 - S1 -> 10^4 m³
		// Convert RHS to be in 10^4 m³ for consistency with S:
		rhs = (S1*10000.0 - (O1 / 2.0 * dtSeconds) + (avgInflow * dtSeconds)) / 10000.0

		// --- Iterative Solver for Z2 ---
		// Find Z2 such that S(Z2) + Q(Z2)/2 * dt / 10000 = RHS (where RHS is now in 10^4 m³)

		var Z2, S2, O2 float64
		toleranceLevel := 0.001 // Tolerance for water level convergence (m)
		maxIterations := 100    // Increase max iterations
		found := false

		// Define the function to find the root for: f(Z) = S(Z) + Q(Z)/(2*10000) * dt - RHS = 0
		calcLHS := func(levelGuess float64) float64 {
			sGuess := getStorageByLevel(levelGuess, reservoir.StorageCurve)                                                                               // 10^4 m³
			qGuess := getDischargeByLevel(levelGuess, reservoir.DischargeCurve, reservoir.DownstreamSafeDischarge, reservoir.Levels.FloodLimitWaterLevel) // m³/s
			return sGuess + qGuess*dtSeconds/(2.0*10000.0)                                                                                                // LHS in 10^4 m³
		}

		// Simple Bisection or Iterative Search (more robust than fixed step trial)
		// We need bounds for the search. Min/Max levels from curves?
		minLevel := reservoir.StorageCurve[0].Level
		maxLevel := reservoir.StorageCurve[len(reservoir.StorageCurve)-1].Level
		// Extend maxLevel slightly in case solution is outside curve range (extrapolation)
		maxLevel += 5.0 // Add a buffer, adjust as needed

		lowLevel := minLevel
		highLevel := maxLevel
		guessLevel := Z1 // Start near previous level

		// Ensure initial guess is within bounds
		if guessLevel < lowLevel {
			guessLevel = lowLevel
		}
		if guessLevel > highLevel {
			guessLevel = highLevel
		}

		for iter := 0; iter < maxIterations; iter++ {
			lhs := calcLHS(guessLevel)
			diff := lhs - rhs

			// Check convergence based on the change in level or closeness of LHS to RHS
			if math.Abs(diff) < 1e-6 { // Check if LHS is close enough to RHS (in 10^4 m³ units)
				Z2 = guessLevel
				found = true
				break
			}

			// Adjust guess using a simple method (like adjusting bounds in bisection)
			// Or a better solver like Newton-Raphson if derivative is known/approximated.
			// Simple approach: Check if f(lowLevel) and f(guessLevel) have different signs
			if calcLHS(lowLevel)*diff < 0 { // Root is between lowLevel and guessLevel
				highLevel = guessLevel
			} else { // Root is between guessLevel and highLevel
				lowLevel = guessLevel
			}

			// New guess is the midpoint
			newGuessLevel := (lowLevel + highLevel) / 2.0

			// Check level convergence
			if math.Abs(newGuessLevel-guessLevel) < toleranceLevel {
				Z2 = newGuessLevel
				found = true
				break
			}
			guessLevel = newGuessLevel

			// Check if bounds become invalid
			if lowLevel > highLevel {
				fmt.Printf("Warning: Solver bounds invalid at step %d\n", i+1)
				break // Exit if bounds cross
			}
		}

		if !found {
			// Simplify the Printf call to ensure correct string termination
			// fmt.Printf("Warning: Solver did not converge for time step %d (t=%.2f). Using last level guess: %.3f\n", i+1, t2.Sub(inflowHydrograph[0].Time.AsTime()).Hours(), guessLevel)
			fmt.Printf("Warning: Solver did not converge at step %d. Using guess: %.3f\n", i+1, guessLevel) // Simplified and ensured \n
			Z2 = guessLevel                                                                                 // Use the last guess, might be inaccurate
			// Consider returning error: return nil, fmt.Errorf("solver did not converge at step %d", i+1)
		}

		// We have Z2 (level at t2)
		S2 = getStorageByLevel(Z2, reservoir.StorageCurve)
		O2 = getDischargeByLevel(Z2, reservoir.DischargeCurve, reservoir.DownstreamSafeDischarge, reservoir.Levels.FloodLimitWaterLevel)

		// 4. Update State and Record
		currentTime = t2
		currentLevel = Z2
		currentStorage = S2
		currentOutflow = O2
		currentInflow = I2 // Inflow for the end of the period (t2)

		resultPoint := RoutingResultPoint{
			Time:          currentTime,
			WaterLevel:    currentLevel,
			StorageVolume: currentStorage, // Already in 10^4 m³
			Outflow:       currentOutflow,
			Inflow:        currentInflow,
		}
		results = append(results, resultPoint)

		// Update peak values
		if currentLevel > peakWaterLevel {
			peakWaterLevel = currentLevel
			peakWaterLevelTime = currentTime
		}
		if currentOutflow > peakOutflow {
			peakOutflow = currentOutflow
			peakOutflowTime = currentTime
		}
		if currentStorage > maxStorageVolume {
			maxStorageVolume = currentStorage
		}
		// Debug log for each step
		// Ensure both lines of the commented Printf are commented
		// fmt.Printf("Step %d: Time=%.2f hrs, In=%.2f, Level=%.3f, Stor=%.3f, Out=%.2f\n",
		//    i+1, currentTime.Sub(inflowHydrograph[0].Time.AsTime()).Hours(), currentInflow, currentLevel, currentStorage, currentOutflow)

	}

	// 5. Assemble Final Result
	finalResult := &RoutingResult{
		Results:            results,
		PeakWaterLevel:     peakWaterLevel,
		PeakWaterLevelTime: peakWaterLevelTime,
		PeakOutflow:        peakOutflow,
		PeakOutflowTime:    peakOutflowTime,
		MaxStorageVolume:   maxStorageVolume,
	}

	fmt.Printf("Routing finished for %s. Peak Level: %.3f at %s, Peak Outflow: %.3f at %s\n",
		reservoir.Name, peakWaterLevel, peakWaterLevelTime.Format(time.RFC3339), peakOutflow, peakOutflowTime.Format(time.RFC3339))

	return finalResult, nil
}

// (省略其他代码...)
