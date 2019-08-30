package app

import (
	gendiodes "code.cloudfoundry.org/go-diodes"
	"code.cloudfoundry.org/go-loggregator/metrics"
	"code.cloudfoundry.org/loggregator-agent/pkg/diodes"
	egress_v2 "code.cloudfoundry.org/loggregator-agent/pkg/egress/v2"
	v2 "code.cloudfoundry.org/loggregator-agent/pkg/ingress/v2"
	"code.cloudfoundry.org/metrics-discovery/internal/collector"
	"code.cloudfoundry.org/tlsconfig"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"log"
	"net/http"
	"time"
)

type MetricsAgent struct {
	cfg                 Config
	log                 *log.Logger
	metrics             Metrics
	metricsServer       *http.Server
	sourceIdBlacklister SourceIDProvider
}

type SourceIDProvider func() []string

type Metrics interface {
	NewCounter(name string, options ...metrics.MetricOption) metrics.Counter
}

func NewMetricsAgent(cfg Config, sourceIdBlacklister SourceIDProvider, metrics Metrics, log *log.Logger) *MetricsAgent {
	return &MetricsAgent{
		cfg:                 cfg,
		log:                 log,
		metrics:             metrics,
		sourceIdBlacklister: sourceIdBlacklister,
	}
}

func (m *MetricsAgent) Run() {
	envelopeBuffer := m.envelopeDiode()
	go m.startIngressServer(envelopeBuffer)

	promCollector := collector.NewEnvelopeCollector(
		m.metrics,
		collector.WithSourceIDExpiration(m.cfg.MetricsExporter.TimeToLive, m.cfg.MetricsExporter.ExpirationInterval),
		collector.WithDefaultTags(m.cfg.MetricsExporter.DefaultTags),
	)
	go m.startEnvelopeCollection(promCollector, envelopeBuffer)

	m.startMetricsServer(promCollector)
}

func (m *MetricsAgent) envelopeDiode() *diodes.ManyToOneEnvelopeV2 {
	ingressDropped := m.metrics.NewCounter(
		"dropped",
		metrics.WithHelpText("Total number of dropped envelopes."),
		metrics.WithMetricTags(map[string]string{"direction": "ingress"}),
	)
	return diodes.NewManyToOneEnvelopeV2(10000, gendiodes.AlertFunc(func(missed int) {
		ingressDropped.Add(float64(missed))
	}))
}

func (m *MetricsAgent) startIngressServer(diode *diodes.ManyToOneEnvelopeV2) {
	ingressMetric := m.metrics.NewCounter(
		"ingress",
		metrics.WithHelpText("Total number of envelopes ingressed by the agent."),
	)
	originMetric := m.metrics.NewCounter(
		"origin_mappings",
		metrics.WithHelpText("Total number of envelopes where the origin tag is used as the source_id."),
	)

	receiver := v2.NewReceiver(diode, ingressMetric, originMetric)
	tlsConfig := m.generateServerTLSConfig(m.cfg.GRPC.CertFile, m.cfg.GRPC.KeyFile, m.cfg.GRPC.CAFile)
	server := v2.NewServer(
		fmt.Sprintf("127.0.0.1:%d", m.cfg.GRPC.Port),
		receiver,
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)

	server.Start()
}

func (m *MetricsAgent) generateServerTLSConfig(certFile, keyFile, caFile string) *tls.Config {
	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(certFile, keyFile),
	).Server(
		tlsconfig.WithClientAuthenticationFromFile(caFile),
	)
	if err != nil {
		log.Fatalf("unable to generate server TLS Config: %s", err)
	}

	return tlsConfig
}

func (m *MetricsAgent) startEnvelopeCollection(promCollector *collector.EnvelopeCollector, diode *diodes.ManyToOneEnvelopeV2) {
	tagger := egress_v2.NewTagger(m.cfg.Tags).TagEnvelope
	timerTagFilterer := egress_v2.NewTimerTagFilterer(m.cfg.MetricsExporter.WhitelistedTimerTags, tagger).Filter
	envelopeWriter := egress_v2.NewEnvelopeWriter(
		promCollector,
		egress_v2.NewCounterAggregator(
			timerTagFilterer,
		),
	)

	for {
		next := diode.Next()
		if m.blacklistedSourceID(next.GetSourceId()) {
			continue
		}

		err := envelopeWriter.Write(next)
		if err != nil {
			log.Printf("unable to write envelope: %s", err)
		}
	}
}

func (m *MetricsAgent) startMetricsServer(promCollector *collector.EnvelopeCollector) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(promCollector)

	router := http.NewServeMux()
	router.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{ErrorHandling: promhttp.ContinueOnError}))

	tlsConfig := m.generateServerTLSConfig(
		m.cfg.MetricsServer.CertFile,
		m.cfg.MetricsServer.KeyFile,
		m.cfg.MetricsServer.CAFile,
	)
	m.metricsServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", m.cfg.MetricsExporter.Port),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		Handler:      router,
		TLSConfig:    tlsConfig,
	}

	log.Printf("Metrics server closing: %s", m.metricsServer.ListenAndServeTLS("", ""))
}

func (m *MetricsAgent) Stop() {
	ctx, cancelFunc := context.WithDeadline(context.Background(), time.Now().Add(15*time.Second))

	go func() {
		defer cancelFunc()

		if m.metricsServer != nil {
			m.metricsServer.Shutdown(ctx)
		}
	}()

	<-ctx.Done()
}

func (m *MetricsAgent) blacklistedSourceID(sourceID string) bool {
	for _, blacklistedSourceID := range m.sourceIdBlacklister() {
		if blacklistedSourceID == sourceID {
			return true
		}
	}

	return false
}
