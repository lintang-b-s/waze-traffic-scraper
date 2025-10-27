package http

import (
	"context"

	http_router "github.com/lintang-b-s/waze-traffic-scraper/pkg/http/router"
	"github.com/lintang-b-s/waze-traffic-scraper/pkg/http/router/controllers"
	http_server "github.com/lintang-b-s/waze-traffic-scraper/pkg/http/server"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	Log *zap.Logger
}

func NewServer(log *zap.Logger) *Server {
	return &Server{Log: log}
}

func (s *Server) Use(
	ctx context.Context,
	log *zap.Logger,

	useRateLimit bool,
	trafficService controllers.TrafficService,

) (*Server, error) {
	viper.SetDefault("API_PORT", 6064)

	viper.SetDefault("API_TIMEOUT", "1000s")

	config := http_server.Config{
		Port:    viper.GetInt("API_PORT"),
		Timeout: viper.GetDuration("API_TIMEOUT"),
	}

	server := http_router.NewAPI(log)

	g := errgroup.Group{}

	g.Go(func() error {
		return server.Run(
			ctx, config, log,
			useRateLimit, trafficService,
		)
	})

	return s, nil
}
