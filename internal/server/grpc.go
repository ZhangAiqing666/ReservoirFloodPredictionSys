package server

import (
	"ReservoirFloodPrediction/internal/conf"
	"ReservoirFloodPrediction/internal/service"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, mapdata *service.MapDataService, inflow *service.InflowService, routing *service.RoutingService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	// mapdataV1.RegisterMapDataServiceServer(srv, mapdata) // 暂时注释，如果你没有实现或生成 gRPC 服务端
	// inflowV1.RegisterInflowServiceServer(srv, inflow)   // 暂时注释
	// routingV1.RegisterRoutingServiceServer(srv, routing) // 暂时注释
	return srv
}
