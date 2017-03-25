package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/rest"

	"github.com/rossf7/pingdom-operator/pkg/pingdom"
)

var (
	log = logging.MustGetLogger("cmd")
)

func Main() int {
	log.Debug("starting operator with k8s v2.0.0")

	// For now always use the built in service account.
	cfg, err := rest.InClusterConfig()
	if err != nil {
		log.Errorf("Error getting Kubernetes config: %v", err)
		return 1
	}

	po, err := pingdom.New(cfg)
	if err != nil {
		log.Errorf("Failed to create pingdom operator: %v", err)
		return 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error { return po.Run(ctx.Done()) })

	term := make(chan os.Signal)
	signal.Notify(term, os.Interrupt, syscall.SIGTERM)

	select {
	case <-term:
		log.Info("Received SIGTERM")
	case <-ctx.Done():
	}

	cancel()
	if err := wg.Wait(); err != nil {
		log.Errorf("Unhanded error exiting: %v", err)
		return 1
	}

	return 0
}

func main() {
	os.Exit(Main())
}
