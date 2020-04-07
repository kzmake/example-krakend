package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
	"github.com/devopsfaith/krakend/router"
	"github.com/devopsfaith/krakend/transport/http/client"

	kgin "github.com/devopsfaith/krakend/router/gin"
	"github.com/gin-gonic/gin"

	spew "github.com/devopsfaith/krakend-spew"
	spewhttp "github.com/devopsfaith/krakend-spew/http"
	ratelimit "github.com/schibsted/krakend-ratelimit"
)

func main() {
	port := flag.Int("p", 0, "Port of the service")
	output := flag.String("o", ".", "Output folder")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("d", false, "Enable the debug")
	configFile := flag.String("c", "/etc/krakend/conf.json", "Path to the configuration filename")
	flag.Parse()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	df := spew.NewFileDumperFactory(ctx, *output, logger)

	// spew http client factory wrapper
	cf := spewhttp.ClientFactory(logger, client.NewHTTPClient, df)
	// spew backend proxy wrapper
	bf := spew.BackendFactory(logger, proxy.CustomHTTPProxyFactory(cf), df)
	// spew proxy wrapper
	pf := spew.ProxyFactory(logger, proxy.NewDefaultFactory(bf, logger), df)
	// spew router wrapper
	runServer := spewhttp.RunServer(logger, router.RunServer, df)

	rateLimitCfg := ratelimit.ConfigGetter(serviceConfig.ExtraConfig).(ratelimit.RateLimitConfig)
	rateLimiter, err := ratelimit.GinRateLimit(rateLimitCfg, ratelimit.DefaultNodeCounter(), logger)
	if err != nil {
		panic(err)
	}

	contextRateLimiter := ratelimit.GinContextRateLimit(rateLimiter, "UserKey")
	middlewares := []gin.HandlerFunc{contextRateLimiter.RateLimit()}

	routerFactory := kgin.NewFactory(kgin.Config{
		Engine:         gin.Default(),
		Middlewares:    middlewares,
		ProxyFactory:   pf,
		Logger:         logger,
		HandlerFactory: kgin.EndpointHandler,
		RunServer:      kgin.RunServerFunc(runServer),
	})

	go func() {
		select {
		case sig := <-sigs:
			logger.Info("Signal intercepted:", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	routerFactory.NewWithContext(ctx).Run(serviceConfig)
}
