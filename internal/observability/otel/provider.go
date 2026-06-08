package otel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"

	obsconfig "github.com/ash-repwiki/ash/internal/observability/config"
)

const defaultServiceName = "ash-worker"

var (
	mu       sync.RWMutex
	provider *sdktrace.TracerProvider
	active   bool
	svcName  = defaultServiceName
)

// Init configures the global OTel tracer provider (noop when disabled).
func Init(cfg *obsconfig.OtelPlugin, serviceName string) (shutdown func(context.Context) error, err error) {
	effective := resolveOtelConfig(cfg)
	if serviceName == "" {
		serviceName = defaultServiceName
	}
	svcName = serviceName

	if effective == nil || !effective.Enabled {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		otel.SetTextMapPropagator(propagation.TraceContext{})
		mu.Lock()
		active = false
		provider = nil
		mu.Unlock()
		return func(context.Context) error { return nil }, nil
	}
	if strings.TrimSpace(effective.Endpoint) == "" {
		return nil, fmt.Errorf("otel.endpoint is required when otel.enabled is true")
	}

	ctx := context.Background()
	opts := []otlptracegrpc.Option{
		otlptracegrpc.WithEndpoint(strings.TrimSpace(effective.Endpoint)),
	}
	if effective.Insecure {
		opts = append(opts, otlptracegrpc.WithInsecure())
	}
	exp, err := otlptracegrpc.New(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("otlp trace exporter: %w", err)
	}

	res, err := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			"",
			attribute.String("service.name", serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("otel resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	mu.Lock()
	active = true
	provider = tp
	mu.Unlock()

	return tp.Shutdown, nil
}

func resolveOtelConfig(cfg *obsconfig.OtelPlugin) *obsconfig.OtelPlugin {
	if strings.TrimSpace(os.Getenv("ASH_OTEL_ENABLED")) == "1" {
		out := &obsconfig.OtelPlugin{Enabled: true, Insecure: true}
		if ep := strings.TrimSpace(os.Getenv("ASH_OTEL_ENDPOINT")); ep != "" {
			out.Endpoint = ep
		}
		if cfg != nil {
			if out.Endpoint == "" {
				out.Endpoint = cfg.Endpoint
			}
			if !cfg.Insecure {
				out.Insecure = cfg.Insecure
			}
		}
		return out
	}
	return cfg
}

// Enabled reports whether OTLP export is active.
func Enabled() bool {
	mu.RLock()
	defer mu.RUnlock()
	return active
}

// ServiceName returns the configured OTel service name.
func ServiceName() string {
	mu.RLock()
	defer mu.RUnlock()
	if svcName == "" {
		return defaultServiceName
	}
	return svcName
}

// Tracer returns a named tracer from the global provider.
func Tracer(name string) trace.Tracer {
	if name == "" {
		name = "github.com/ash-repwiki/ash/otel"
	}
	return otel.Tracer(name)
}

// Status summarizes runtime OTel wiring for ops endpoints.
type Status struct {
	Enabled     bool   `json:"enabled"`
	ServiceName string `json:"serviceName"`
	Endpoint    string `json:"endpoint,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`
}

// RuntimeStatus returns the active OTel export configuration.
func RuntimeStatus(cfg *obsconfig.OtelPlugin) Status {
	effective := resolveOtelConfig(cfg)
	st := Status{Enabled: Enabled(), ServiceName: ServiceName()}
	if effective != nil {
		st.Endpoint = effective.Endpoint
		st.Insecure = effective.Insecure
	}
	return st
}
