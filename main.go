//go:generate go run pkg/codegen/cleanup/main.go
//go:generate go run pkg/codegen/main.go

package main

import (
	"flag"
	"os"

	nested "github.com/antonfisher/nested-logrus-formatter"
	"github.com/cnrancher/cce-operator/pkg/controller"
	ccev1 "github.com/cnrancher/cce-operator/pkg/generated/controllers/cce.pandaria.io"
	"github.com/cnrancher/cce-operator/pkg/utils"
	corev1 "github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/rancher/wrangler/pkg/start"
	"github.com/sirupsen/logrus"
)

var (
	masterURL      string
	kubeconfigFile string
	version        bool
	debug          bool
)

func init() {
	logrus.SetFormatter(&nested.Formatter{
		HideKeys:        true,
		TimestampFormat: "2006-01-02 15:04:05",
		FieldsOrder:     []string{"cluster", "phase"},
	})

	flag.StringVar(&kubeconfigFile, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.BoolVar(&version, "version", false, "Show version.")
	flag.BoolVar(&debug, "debug", false, "Enable the debug output.")
	flag.Parse()

	if debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("debug output enabled")
	}
	if version {
		if utils.GitCommit != "" {
			logrus.Infof("cce-operator %v - %v", utils.Version, utils.GitCommit)
		} else {
			logrus.Infof("cce-operator %v", utils.Version)
		}
		os.Exit(0)
	}
}

func main() {
	// set up signals so we handle the first shutdown signal gracefully
	ctx := signals.SetupSignalContext()

	// This will load the kubeconfig file in a style the same as kubectl
	cfg, err := kubeconfig.GetNonInteractiveClientConfig(kubeconfigFile).ClientConfig()
	if err != nil {
		logrus.Fatalf("Error building kubeconfig: %v", err)
	}

	// Generated controller
	core := corev1.NewFactoryFromConfigOrDie(cfg)
	cce := ccev1.NewFactoryFromConfigOrDie(cfg)

	// The typical pattern is to build all your controller/clients then just pass to each handler
	// the bare minimum of what they need.  This will eventually help with writing tests.  So
	// don't pass in something like kubeClient, apps, or sample
	controller.Register(ctx,
		core.Core().V1().Secret(),
		cce.Cce().V1().CCEClusterConfig())

	// Start all the controllers
	if err := start.All(ctx, 2, cce, core); err != nil {
		logrus.Fatalf("Error starting cce controller: %v", err)
	}

	<-ctx.Done()
	logrus.Infof("CCE Operator stopped gracefully")
}
