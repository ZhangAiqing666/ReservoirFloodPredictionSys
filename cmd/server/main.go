package main

import (
	"flag"
	"os"

	"ReservoirFloodPrediction/internal/conf"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name = "ReservoirFloodPrediction" // 应用名称
	// Version is the version of the compiled software.
	Version string

	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

// newApp creates a new Kratos application.
// 它接收 Logger 和 HTTP Server 作为参数 (可以按需添加 gRPC Server)
func newApp(logger log.Logger, hs *http.Server) *kratos.App { // 确保 hs 参数类型正确
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs, // 添加 HTTP Server
			// rs, // 如果有 gRPC Server 也添加进来
		),
	)
}

func main() {
	flag.Parse()
	logger := log.NewStdLogger(os.Stdout)
	logger = log.With(
		logger,
		"ts",
		log.DefaultTimestamp,
		"caller",
		log.DefaultCaller,
		"service.id",
		id,
		"service.name",
		Name,
		"service.version",
		Version,
		"trace.id",
		tracing.TraceID(),
		"span.id",
		tracing.SpanID(),
	)
	c := config.New(
		config.WithSource(file.NewSource(flagconf)),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		panic(err)
	}

	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		panic(err)
	}

	// 注入依赖并启动应用
	app, cleanup, err := wireApp(&bc, logger)
	if err != nil {
		panic(err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
