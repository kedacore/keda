/*
Copyright 2018 The Knative Authors

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
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/knative/pkg/signals"

	"github.com/knative/pkg/logging/logkey"
	"github.com/knative/pkg/websocket"
	"github.com/knative/serving/cmd/util"
	activatorutil "github.com/knative/serving/pkg/activator/util"
	"github.com/knative/serving/pkg/apis/serving/v1alpha1"
	"github.com/knative/serving/pkg/autoscaler"
	"github.com/knative/serving/pkg/http/h2c"
	"github.com/knative/serving/pkg/logging"
	"github.com/knative/serving/pkg/network"
	"github.com/knative/serving/pkg/queue"
	"github.com/knative/serving/pkg/queue/health"
	"github.com/knative/serving/pkg/utils"
	"go.opencensus.io/exporter/prometheus"
	"go.opencensus.io/stats/view"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	// Add a little buffer space between request handling and stat
	// reporting so that latency in the stat pipeline doesn't
	// interfere with request handling.
	statReportingQueueLength = 10

	// Add enough buffer to not block request serving on stats collection
	requestCountingQueueLength = 100

	// Duration the /quitquitquit handler should wait before returning.
	// This is to give Istio a little bit more time to remove the pod
	// from its configuration and propagate that to all istio-proxies
	// in the mesh.
	quitSleepDuration = 20 * time.Second

	// Only report errors about a non-existent websocket connection after
	// having been up and running for this long.
	startupConnectionGrace = 10 * time.Second
)

var (
	podName                string
	servingConfig          string
	servingNamespace       string
	servingRevision        string
	servingRevisionKey     string
	servingAutoscaler      string
	autoscalerNamespace    string
	servingAutoscalerPort  int
	userTargetPort         int
	userTargetAddress      string
	containerConcurrency   int
	revisionTimeoutSeconds int
	statChan               = make(chan *autoscaler.Stat, statReportingQueueLength)
	reqChan                = make(chan queue.ReqEvent, requestCountingQueueLength)
	statSink               *websocket.ManagedConnection
	logger                 *zap.SugaredLogger
	breaker                *queue.Breaker

	h2cProxy  *httputil.ReverseProxy
	httpProxy *httputil.ReverseProxy

	server      *http.Server
	healthState = &health.State{}
	reporter    *queue.Reporter // Prometheus stats reporter.

	startupTime = time.Now()
)

func initEnv() {
	podName = util.GetRequiredEnvOrFatal("SERVING_POD", logger)
	servingConfig = util.GetRequiredEnvOrFatal("SERVING_CONFIGURATION", logger)
	servingNamespace = util.GetRequiredEnvOrFatal("SERVING_NAMESPACE", logger)
	servingRevision = util.GetRequiredEnvOrFatal("SERVING_REVISION", logger)
	servingAutoscaler = util.GetRequiredEnvOrFatal("SERVING_AUTOSCALER", logger)
	autoscalerNamespace = util.GetRequiredEnvOrFatal("SYSTEM_NAMESPACE", logger)
	servingAutoscalerPort = util.MustParseIntEnvOrFatal("SERVING_AUTOSCALER_PORT", logger)
	containerConcurrency = util.MustParseIntEnvOrFatal("CONTAINER_CONCURRENCY", logger)
	revisionTimeoutSeconds = util.MustParseIntEnvOrFatal("REVISION_TIMEOUT_SECONDS", logger)
	userTargetPort = util.MustParseIntEnvOrFatal("USER_PORT", logger)
	userTargetAddress = fmt.Sprintf("127.0.0.1:%d", userTargetPort)

	// TODO(mattmoor): Move this key to be in terms of the KPA.
	servingRevisionKey = autoscaler.NewMetricKey(servingNamespace, servingRevision)
	_reporter, err := queue.NewStatsReporter(servingNamespace, servingConfig, servingRevision, podName)
	if err != nil {
		logger.Fatalw("Failed to create stats reporter", zap.Error(err))
	}
	reporter = _reporter
}

func statReporter() {
	for {
		s := <-statChan
		if err := sendStat(s); err != nil {
			// Hide "not-established" errors until the startupConnectionGrace has passed.
			if err != websocket.ErrConnectionNotEstablished || time.Since(startupTime) > startupConnectionGrace {
				logger.Errorw("Error while sending stat", zap.Error(err))
			}
		}
	}
}

// sendStat sends a single StatMessage to the autoscaler.
func sendStat(s *autoscaler.Stat) error {
	if statSink == nil {
		return errors.New("stat sink not (yet) connected")
	}
	reporter.Report(
		float64(s.RequestCount),
		float64(s.AverageConcurrentRequests),
	)
	if healthState.IsShuttingDown() {
		// Do not send metrics if the pods is shutting down.
		return nil
	}
	sm := autoscaler.StatMessage{
		Stat: *s,
		Key:  servingRevisionKey,
	}
	return statSink.Send(sm)
}

func proxyForRequest(req *http.Request) *httputil.ReverseProxy {
	if req.ProtoMajor == 2 {
		return h2cProxy
	}

	return httpProxy
}

func isKnativeProbe(r *http.Request) bool {
	return r.Header.Get(network.ProbeHeaderName) != ""
}

func isKubeletProbe(r *http.Request) bool {
	// Since K8s 1.8, prober requests have
	//   User-Agent = "kube-probe/{major-version}.{minor-version}".
	return strings.HasPrefix(r.Header.Get("User-Agent"), "kube-probe/")
}

func probeUserContainer() bool {
	var err error
	wait.PollImmediate(50*time.Millisecond, 10*time.Second, func() (bool, error) {
		logger.Debug("TCP probing the user-container.")
		err = health.TCPProbe(userTargetAddress, 100*time.Millisecond)
		return err == nil, nil
	})

	if err == nil {
		logger.Info("User-container successfully probed.")
	} else {
		logger.Errorw("User-container could not be probed successfully.", zap.Error(err))
	}

	return err == nil
}

func handler(w http.ResponseWriter, r *http.Request) {
	proxy := proxyForRequest(r)

	switch {
	case isKnativeProbe(r):
		if probeUserContainer() {
			// Respond with the name of the component handling the request.
			w.Write([]byte("queue"))
		} else {
			http.Error(w, "container not ready", http.StatusServiceUnavailable)
		}
		return

	case isKubeletProbe(r):
		// Do not count health checks for concurrency metrics
		proxy.ServeHTTP(w, r)
		return
	}

	// Metrics for autoscaling
	reqChan <- queue.ReqEvent{Time: time.Now(), EventType: queue.ReqIn}
	defer func() {
		reqChan <- queue.ReqEvent{Time: time.Now(), EventType: queue.ReqOut}
	}()

	// Enforce queuing and concurrency limits
	if breaker != nil {
		ok := breaker.Maybe(func() {
			proxy.ServeHTTP(w, r)
		})
		if !ok {
			http.Error(w, "overload", http.StatusServiceUnavailable)
		}
	} else {
		proxy.ServeHTTP(w, r)
	}

}

// Sets up /health and /quitquitquit endpoints.
func createAdminHandlers() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc(queue.RequestQueueHealthPath, healthState.HealthHandler(probeUserContainer))
	mux.HandleFunc(queue.RequestQueueQuitPath, healthState.QuitHandler(func() {
		time.Sleep(quitSleepDuration)

		// Shutdown the proxy server.
		if server != nil {
			if err := server.Shutdown(context.Background()); err != nil {
				logger.Errorw("Failed to shutdown proxy-server", zap.Error(err))
			} else {
				logger.Debug("Proxy server shutdown successfully.")
			}
		}
	}))

	return mux
}

func main() {
	flag.Parse()
	logger, _ = logging.NewLogger(os.Getenv("SERVING_LOGGING_CONFIG"), os.Getenv("SERVING_LOGGING_LEVEL"))
	logger = logger.Named("queueproxy")
	defer logger.Sync()

	initEnv()
	logger = logger.With(
		zap.String(logkey.Key, servingRevisionKey),
		zap.String(logkey.Pod, podName))

	target, err := url.Parse(fmt.Sprintf("http://%s", userTargetAddress))
	if err != nil {
		logger.Fatalw("Failed to parse localhost url", zap.Error(err))
	}

	httpProxy = httputil.NewSingleHostReverseProxy(target)
	httpProxy.FlushInterval = -1
	h2cProxy = httputil.NewSingleHostReverseProxy(target)
	h2cProxy.Transport = h2c.DefaultTransport
	h2cProxy.FlushInterval = -1

	activatorutil.SetupHeaderPruning(httpProxy)
	activatorutil.SetupHeaderPruning(h2cProxy)

	// If containerConcurrency == 0 then concurrency is unlimited.
	if containerConcurrency > 0 {
		// We set the queue depth to be equal to the container concurrency but at least 10 to
		// allow the autoscaler to get a strong enough signal.
		queueDepth := containerConcurrency
		if queueDepth < 10 {
			queueDepth = 10
		}
		params := queue.BreakerParams{QueueDepth: int32(queueDepth), MaxConcurrency: int32(containerConcurrency), InitialCapacity: int32(containerConcurrency)}
		breaker = queue.NewBreaker(params)
		logger.Infof("Queue container is starting with %#v", params)
	}

	logger.Info("Initializing OpenCensus Prometheus exporter")
	promExporter, err := prometheus.NewExporter(prometheus.Options{Namespace: "queue"})
	if err != nil {
		logger.Fatalw("Failed to create the Prometheus exporter", zap.Error(err))
	}
	view.RegisterExporter(promExporter)
	view.SetReportingPeriod(queue.ViewReportingPeriod)
	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promExporter)
		http.ListenAndServe(fmt.Sprintf(":%d", v1alpha1.RequestQueueMetricsPort), mux)
	}()

	// Open a websocket connection to the autoscaler
	autoscalerEndpoint := fmt.Sprintf("ws://%s.%s.svc.%s:%d", servingAutoscaler, autoscalerNamespace, utils.GetClusterDomainName(), servingAutoscalerPort)
	logger.Infof("Connecting to autoscaler at %s", autoscalerEndpoint)
	statSink = websocket.NewDurableSendingConnection(autoscalerEndpoint, logger)
	go statReporter()

	reportTicker := time.NewTicker(queue.ReporterReportingPeriod).C
	queue.NewStats(podName, queue.Channels{
		ReqChan:    reqChan,
		ReportChan: reportTicker,
		StatChan:   statChan,
	}, time.Now())

	adminServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", v1alpha1.RequestQueueAdminPort),
		Handler: nil,
	}
	adminServer.Handler = createAdminHandlers()

	server = h2c.NewServer(
		fmt.Sprintf(":%d", v1alpha1.RequestQueuePort),
		queue.TimeToFirstByteTimeoutHandler(http.HandlerFunc(handler), time.Duration(revisionTimeoutSeconds)*time.Second, "request timeout"))

	errChan := make(chan error, 2)
	// Runs a server created by creator and sends fatal errors to the errChan.
	// Does not act on the ErrServerClosed error since that indicates we're
	// already shutting everything down.
	catchServerError := func(creator func() error) {
		if err := creator(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}

	go catchServerError(server.ListenAndServe)
	go catchServerError(adminServer.ListenAndServe)

	// Blocks until we actually receive a TERM signal or one of the servers
	// exit unexpectedly. We fold both signals together because we only want
	// to act on the first of those to reach here.
	select {
	case err := <-errChan:
		logger.Errorw("Failed to bring up queue-proxy, shutting down.", zap.Error(err))
		os.Exit(1)
	case <-signals.SetupSignalHandler():
		logger.Info("Received TERM signal, attempting to gracefully shutdown servers.")

		// Calling server.Shutdown() allows pending requests to
		// complete, while no new work is accepted.
		if err := adminServer.Shutdown(context.Background()); err != nil {
			logger.Errorw("Failed to shutdown admin-server", zap.Error(err))
		}

		if statSink != nil {
			if err := statSink.Shutdown(); err != nil {
				logger.Errorw("Failed to shutdown websocket connection", zap.Error(err))
			}
		}
	}
}
