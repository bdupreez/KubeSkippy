/*
Copyright 2024 The KubeSkippy Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/client-go/kubernetes"
	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned"
	kubeskippyv1alpha1 "github.com/kubeskippy/kubeskippy/api/v1alpha1"
	"github.com/kubeskippy/kubeskippy/internal/controller"
	kubemetrics "github.com/kubeskippy/kubeskippy/internal/metrics"
	"github.com/kubeskippy/kubeskippy/internal/remediation"
	"github.com/kubeskippy/kubeskippy/internal/safety"
	"github.com/kubeskippy/kubeskippy/pkg/config"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kubeskippyv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var watchNamespace string
	var dryRun bool

	flag.StringVar(&configFile, "config", "", "The controller config file")
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&watchNamespace, "namespace", "", "Namespace to watch (empty means all namespaces)")
	flag.BoolVar(&dryRun, "dry-run", false, "Run in dry-run mode (no actual healing actions)")
	
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Load configuration
	cfg := config.NewDefaultConfig()
	if configFile != "" {
		// TODO: Load config from file
		setupLog.Info("Loading config from file", "path", configFile)
	}

	// Override with command line flags
	if metricsAddr != "" {
		cfg.MetricsAddr = metricsAddr
	}
	if probeAddr != "" {
		cfg.ProbeAddr = probeAddr
	}
	cfg.EnableLeaderElection = enableLeaderElection
	cfg.WatchNamespace = watchNamespace
	if dryRun {
		cfg.Safety.DryRunMode = true
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		setupLog.Error(err, "Invalid configuration")
		os.Exit(1)
	}

	// Create manager options
	mgrOpts := ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: cfg.ProbeAddr,
		LeaderElection:         cfg.EnableLeaderElection,
		LeaderElectionID:       "kubeskippy.io",
	}
	
	// Create manager
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOpts)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize components
	setupLog.Info("Initializing components")

	// Create safety controller with in-memory store
	safetyStore := safety.NewInMemoryActionStore()
	safetyController := safety.NewController(mgr.GetClient(), cfg.Safety, safetyStore, nil)
	
	// Start cleanup loop for old action records
	ctx := ctrl.SetupSignalHandler()
	safetyController.StartCleanupLoop(ctx, 24*time.Hour)

	// Create Kubernetes clients for metrics collector
	kubeConfig := ctrl.GetConfigOrDie()
	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		setupLog.Error(err, "unable to create kubernetes clientset")
		os.Exit(1)
	}
	
	metricsClientset, err := metricsclient.NewForConfig(kubeConfig)
	if err != nil {
		setupLog.Error(err, "unable to create metrics clientset")
		// Continue without metrics client - metrics server might not be installed
		setupLog.Info("Metrics server not available, some metrics will be unavailable")
	}

	// Create metrics collector
	metricsCollector := kubemetrics.NewCollector(mgr.GetClient(), clientset, metricsClientset)
	
	// Configure Prometheus if enabled
	if cfg.Metrics.PrometheusURL != "" {
		setupLog.Info("Configuring Prometheus integration", "url", cfg.Metrics.PrometheusURL)
		if err := metricsCollector.WithPrometheus(cfg.Metrics.PrometheusURL); err != nil {
			setupLog.Error(err, "Failed to configure Prometheus integration")
			// Continue without Prometheus - it's optional
		} else {
			setupLog.Info("Prometheus integration enabled successfully")
		}
	}
	
	// Create remediation engine with action recorder
	actionRecorder := remediation.NewInMemoryActionRecorder(24 * time.Hour)
	actionRecorder.StartCleanupLoop(ctx, 1*time.Hour)
	remediationEngine := remediation.NewEngine(mgr.GetClient(), actionRecorder)
	
	// TODO: Create actual implementation for AI analyzer
	var aiAnalyzer controller.AIAnalyzer
	
	setupLog.Info("Safety controller, metrics collector, and remediation engine initialized")

	// Setup controllers
	if err = (&controller.HealingPolicyReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		Config:           cfg,
		MetricsCollector: metricsCollector,
		SafetyController: safetyController,
		AIAnalyzer:       aiAnalyzer,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HealingPolicy")
		os.Exit(1)
	}

	if err = (&controller.HealingActionReconciler{
		Client:            mgr.GetClient(),
		Scheme:            mgr.GetScheme(),
		Config:            cfg,
		RemediationEngine: remediationEngine,
		SafetyController:  safetyController,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "HealingAction")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	// Add health checks
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Register custom metrics
	registerMetrics()

	// Start manager
	setupLog.Info("Starting manager", "version", "v0.1.0", "dry-run", cfg.Safety.DryRunMode)
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// registerMetrics registers custom Prometheus metrics
func registerMetrics() {
	// Register healing action metrics
	healingActionsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeskippy_healing_actions_total",
			Help: "Total number of healing actions taken",
		},
		[]string{"action_type", "namespace", "status"},
	)
	metrics.Registry.MustRegister(healingActionsTotal)

	// Register policy evaluation metrics
	policyEvaluationsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeskippy_policy_evaluations_total",
			Help: "Total number of policy evaluations",
		},
		[]string{"policy", "namespace", "result"},
	)
	metrics.Registry.MustRegister(policyEvaluationsTotal)

	// Register AI analysis metrics
	aiAnalysisLatency := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kubeskippy_ai_analysis_duration_seconds",
			Help:    "Latency of AI analysis in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"model", "status"},
	)
	metrics.Registry.MustRegister(aiAnalysisLatency)

	// Register safety validation metrics
	safetyValidationsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kubeskippy_safety_validations_total",
			Help: "Total number of safety validations",
		},
		[]string{"result"},
	)
	metrics.Registry.MustRegister(safetyValidationsTotal)
}