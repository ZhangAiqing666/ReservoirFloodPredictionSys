package server

import (
	"github.com/google/wire"
)

// ProviderSet is server providers.
var ProviderSet = wire.NewSet(NewHTTPServer, NewGRPCServer)

// --- NewHTTPServer 函数体已移至 http.go ---
// --- NewGRPCServer 函数体已移至 grpc.go ---
