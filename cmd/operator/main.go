package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/op/go-logging"
	"golang.org/x/sync/errgroup"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/rest"

	"github.com/rossf7/pingdom-operator/pkg/pingdom"
	"github.com/rossf7/pingdom-operator/pkg/tpr"
)

var (
	log = logging.MustGetLogger("cmd")
)

func Main() int {
	var clientset *kubernetes.Clientset
	{
		// For now always use the built in service account.
		config, err := rest.InClusterConfig()
		if err != nil {
			log.Errorf("Error getting Kubernetes config: %v", err)
			return 1
		}

		clientset, err = kubernetes.NewForConfig(config)
		if err != nil {
			log.Errorf("Error creating Kubernetes clientset: %v", err)
			return 1
		}
	}

	to := tpr.New(v1.NamespaceAll, clientset, tpr.NewStore())
	po := pingdom.New(v1.NamespaceAll, clientset)

	ctx, cancel := context.WithCancel(context.Background())
	wg, ctx := errgroup.WithContext(ctx)

	wg.Go(func() error { return to.Run(ctx.Done()) })
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
