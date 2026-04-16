package service

import (
	"github.com/google/wire"
)

// ProviderSet is service providers.
var ProviderSet = wire.NewSet(
	//NewGreeterService, // 保留现有的 Service (示例)
	NewMapDataService, // 保留现有的 Service
	NewInflowService,  // 保留现有的 Service
	NewRoutingService, // <--- 添加 RoutingService
	NewUserService,    // <--- 添加 UserService
)
