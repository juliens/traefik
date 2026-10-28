package types

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/traefik/traefik/v3/pkg/observability"
	ttypes "github.com/traefik/traefik/v3/pkg/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/encoding/gzip"
)

// TracingVerbosity is defined in pkg/types, so the dynamic configuration does not depend on the OpenTelemetry exporters.
type TracingVerbosity = ttypes.TracingVerbosity

const (
	MinimalVerbosity  = ttypes.MinimalVerbosity
	DetailedVerbosity = ttypes.DetailedVerbosity
)

// OTelTracing provides configuration settings for the open-telemetry tracer.
type OTelTracing struct {
	GRPC *OTelGRPC `description:"gRPC configuration for the OpenTelemetry collector." json:"grpc,omitempty" toml:"grpc,omitempty" yaml:"grpc,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
	HTTP *OTelHTTP `description:"HTTP configuration for the OpenTelemetry collector." json:"http,omitempty" toml:"http,omitempty" yaml:"http,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`
}

// SetDefaults sets the default values.
func (c *OTelTracing) SetDefaults() {
	c.HTTP = &OTelHTTP{}
	c.HTTP.SetDefaults()
}

// Setup sets up the tracer.
func (c *OTelTracing) Setup(ctx context.Context, serviceName string, sampleRate float64, resourceAttributes map[string]string) (trace.Tracer, io.Closer, error) {
	var (
		err      error
		exporter *otlptrace.Exporter
	)
	if c.GRPC != nil {
		exporter, err = c.setupGRPCExporter()
	} else {
		exporter, err = c.setupHTTPExporter()
	}
	if err != nil {
		return nil, nil, fmt.Errorf("setting up exporter: %w", err)
	}

	res, err := observability.NewOTelResource(ctx, serviceName, resourceAttributes)
	if err != nil {
		return nil, nil, err
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(sampleRate))),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	otel.SetTracerProvider(tracerProvider)

	log.Debug().Msg("OpenTelemetry tracer configured")

	return tracerProvider.Tracer("github.com/traefik/traefik"), &tpCloser{provider: tracerProvider}, err
}

func (c *OTelTracing) setupHTTPExporter() (*otlptrace.Exporter, error) {
	endpoint, err := url.Parse(c.HTTP.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.HTTP.Endpoint, err)
	}

	opts := []otlptracehttp.Option{
		otlptracehttp.WithEndpoint(endpoint.Host),
		otlptracehttp.WithHeaders(c.HTTP.Headers),
		otlptracehttp.WithCompression(otlptracehttp.GzipCompression),
	}

	if endpoint.Scheme == "http" {
		opts = append(opts, otlptracehttp.WithInsecure())
	}

	if endpoint.Path != "" {
		opts = append(opts, otlptracehttp.WithURLPath(endpoint.Path))
	}

	if c.HTTP.TLS != nil {
		tlsConfig, err := c.HTTP.TLS.CreateTLSConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("creating TLS client config: %w", err)
		}

		opts = append(opts, otlptracehttp.WithTLSClientConfig(tlsConfig))
	}

	return otlptrace.New(context.Background(), otlptracehttp.NewClient(opts...))
}

func (c *OTelTracing) setupGRPCExporter() (*otlptrace.Exporter, error) {
	host, port, err := net.SplitHostPort(c.GRPC.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid collector endpoint %q: %w", c.GRPC.Endpoint, err)
	}

	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%s", host, port)),
		otlptracegrpc.WithHeaders(c.GRPC.Headers),
		otlptracegrpc.WithCompressor(gzip.Name),
	}

	if c.GRPC.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}

	if c.GRPC.TLS != nil {
		tlsConfig, err := c.GRPC.TLS.CreateTLSConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("creating TLS client config: %w", err)
		}

		opts = append(opts, otlptracegrpc.WithTLSCredentials(credentials.NewTLS(tlsConfig)))
	}

	return otlptrace.New(context.Background(), otlptracegrpc.NewClient(opts...))
}

// tpCloser converts a TraceProvider into an io.Closer.
type tpCloser struct {
	provider *sdktrace.TracerProvider
}

func (t *tpCloser) Close() error {
	if t == nil {
		return nil
	}

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(5*time.Second))
	defer cancel()

	return t.provider.Shutdown(ctx)
}

// Tracing holds the tracing configuration.
type Tracing struct {
	ServiceName             string            `description:"Defines the service name resource attribute." json:"serviceName,omitempty" toml:"serviceName,omitempty" yaml:"serviceName,omitempty" export:"true"`
	ResourceAttributes      map[string]string `description:"Defines additional resource attributes (key:value)." json:"resourceAttributes,omitempty" toml:"resourceAttributes,omitempty" yaml:"resourceAttributes,omitempty" export:"true"`
	CapturedRequestHeaders  []string          `description:"Request headers to add as attributes for server and client spans." json:"capturedRequestHeaders,omitempty" toml:"capturedRequestHeaders,omitempty" yaml:"capturedRequestHeaders,omitempty" export:"true"`
	CapturedResponseHeaders []string          `description:"Response headers to add as attributes for server and client spans." json:"capturedResponseHeaders,omitempty" toml:"capturedResponseHeaders,omitempty" yaml:"capturedResponseHeaders,omitempty" export:"true"`
	SafeQueryParams         []string          `description:"Query params to not redact." json:"safeQueryParams,omitempty" toml:"safeQueryParams,omitempty" yaml:"safeQueryParams,omitempty" export:"true"`
	SampleRate              float64           `description:"Sets the rate between 0.0 and 1.0 of requests to trace." json:"sampleRate,omitempty" toml:"sampleRate,omitempty" yaml:"sampleRate,omitempty" export:"true"`
	AddInternals            bool              `description:"Enables tracing for internal services (ping, dashboard, etc...)." json:"addInternals,omitempty" toml:"addInternals,omitempty" yaml:"addInternals,omitempty" export:"true"`
	OTLP                    *OTelTracing      `description:"Settings for OpenTelemetry." json:"otlp,omitempty" toml:"otlp,omitempty" yaml:"otlp,omitempty" label:"allowEmpty" file:"allowEmpty" export:"true"`

	// Deprecated: please use ResourceAttributes instead.
	GlobalAttributes map[string]string `description:"(Deprecated) Defines additional resource attributes (key:value)." json:"globalAttributes,omitempty" toml:"globalAttributes,omitempty" yaml:"globalAttributes,omitempty" export:"true"`
}

// SetDefaults sets the default values.
func (t *Tracing) SetDefaults() {
	t.ServiceName = OTelTraefikServiceName
	t.SampleRate = 1.0

	t.OTLP = &OTelTracing{}
	t.OTLP.SetDefaults()
}
