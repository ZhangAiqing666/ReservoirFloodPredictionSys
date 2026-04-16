package data

import (
	"ReservoirFloodPrediction/internal/biz"

	"github.com/go-kratos/kratos/v2/log"
)

// inflowRepo 结构体实现了 biz.InflowRepo 接口
type inflowRepo struct {
	data *Data // 依赖 Data
	log  *log.Helper
}

// NewInflowRepo 创建一个新的 inflowRepo
// 它实现了 biz.InflowRepo 接口
func NewInflowRepo(data *Data, logger log.Logger) biz.InflowRepo { // 返回 biz.InflowRepo 接口
	return &inflowRepo{
		data: data,
		log:  log.NewHelper(log.With(logger, "module", "data/inflow")),
	}
}

// 未来可以在这里实现 InflowRepo 接口中定义的方法
// 例如：
// func (r *inflowRepo) SaveSimulationData(ctx context.Context, data ...) error {
//     // ... 实现保存逻辑 ...
// }
