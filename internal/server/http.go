package server

import (
	inflowv1 "ReservoirFloodPrediction/api/inflow/v1"
	mapdatav1 "ReservoirFloodPrediction/api/mapdata/v1"
	routingV1 "ReservoirFloodPrediction/api/routing/v1"
	userV1 "ReservoirFloodPrediction/api/user/v1"

	// userV1 "ReservoirFloodPrediction/api/user/v1" // 移除 User API 导入
	"ReservoirFloodPrediction/internal/conf"
	"ReservoirFloodPrediction/internal/service"

	// 添加 context 包导入
	// 导入 net/http
	"github.com/go-kratos/kratos/v2/log"                 // 导入 middleware
	"github.com/go-kratos/kratos/v2/middleware/recovery" // 导入 transport
	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"

	"github.com/gorilla/handlers" // 1. 导入 gorilla/handlers
)

// corsMiddleware 函数及其相关的注释已被彻底删除

// NewHTTPServer new an HTTP server.
func NewHTTPServer(c *conf.Server, user *service.UserService, mapdata *service.MapDataService, inflow *service.InflowService, routing *service.RoutingService, logger log.Logger) *kratosHttp.Server {

	// !!! 重要: 确保端口号与前端一致 !!!
	allowedOrigins := handlers.AllowedOrigins([]string{"http://localhost:8081"})
	allowedMethods := handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	allowedHeaders := handlers.AllowedHeaders([]string{"Content-Type", "Authorization"})
	// allowCredentials := handlers.AllowCredentials() // 如果需要 Cookie 支持，取消注释

	var opts = []kratosHttp.ServerOption{
		// 使用 kratosHttp.Filter 集成 gorilla/handlers CORS 中间件
		kratosHttp.Filter(handlers.CORS(allowedOrigins, allowedMethods, allowedHeaders /*, allowCredentials */)),
		kratosHttp.Middleware(
			recovery.Recovery(),
			// 不再需要 corsMiddleware()
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, kratosHttp.Network(c.Http.Network))
	}
	if c.Http.Addr != "" {
		opts = append(opts, kratosHttp.Address(c.Http.Addr))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, kratosHttp.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := kratosHttp.NewServer(opts...)
	userV1.RegisterUserServiceHTTPServer(srv, user)
	mapdatav1.RegisterMapDataHTTPServer(srv, mapdata)
	inflowv1.RegisterInflowServiceHTTPServer(srv, inflow)
	routingV1.RegisterRoutingServiceHTTPServer(srv, routing)
	return srv
}
