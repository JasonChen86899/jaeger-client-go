// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config_test

import (
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-lib/metrics"

	"github.com/uber/jaeger-client-go"
	jaegercfg "github.com/uber/jaeger-client-go/config"
	jaegerlog "github.com/uber/jaeger-client-go/log"
)

func ExampleFromEnv() {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)
	// continue main()
}

func ExampleFromEnv_override() {
	os.Setenv("JAEGER_SERVICE_NAME", "not-effective")

	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	cfg.ServiceName = "this-will-be-the-service-name"

	tracer, closer, err := cfg.NewTracer()
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	opentracing.SetGlobalTracer(tracer)
	// continue main()
}

func ExampleConfiguration_InitGlobalTracer_testing() {
	// Sample configuration for testing. Use constant sampling to sample every trace
	// and enable LogSpan to log every span via configured Logger.
	cfg := jaegercfg.Configuration{
		Sampler: &jaegercfg.SamplerConfig{
			Type:  jaeger.SamplerTypeConst,
			Param: 1,
		},
		Reporter: &jaegercfg.ReporterConfig{
			LogSpans: true,
		},
	}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	// continue main()
}

func ExampleConfiguration_InitGlobalTracer_production() {
	// Recommended configuration for production.
	cfg := jaegercfg.Configuration{}

	// Example logger and metrics factory. Use github.com/uber/jaeger-client-go/log
	// and github.com/uber/jaeger-lib/metrics respectively to bind to real logging and metrics
	// frameworks.
	jLogger := jaegerlog.StdLogger
	jMetricsFactory := metrics.NullFactory

	// Initialize tracer with a logger and a metrics factory
	closer, err := cfg.InitGlobalTracer(
		"serviceName",
		jaegercfg.Logger(jLogger),
		jaegercfg.Metrics(jMetricsFactory),
	)
	if err != nil {
		log.Printf("Could not initialize jaeger tracer: %s", err.Error())
		return
	}
	defer closer.Close()

	// continue main()
}

func TestAllInOne(t *testing.T) {
	cfg, err := jaegercfg.FromEnv()
	if err != nil {
		// parsing errors might happen here, such as when we get a string where we expect a number
		log.Printf("Could not parse Jaeger env vars: %s", err.Error())
		return
	}

	cfg.ServiceName = "this-will-be-the-service-name"

	sender, err := jaeger.NewUDPTransport("127.0.0.1:6831", 1000)
	if err != nil {
		fmt.Println("udp error")
		return
	}

	rOps := jaeger.ReporterOptions
	reporter := jaeger.NewRemoteReporter(sender, rOps.BufferFlushInterval(time.Second))
	// Initialize tracer with a logger and a metrics factory
	tracer, _, err := cfg.NewTracer(
		jaegercfg.Reporter(reporter),
	)

	if err !=  nil {
		log.Println(err)
		return
	}

	opentracing.SetGlobalTracer(tracer)

	for {
		for i := 0; i < 10000; i++ {
			spanContext := jaeger.NewSpanContext(jaeger.TraceID{
				High: 9,
				Low:  9,
			}, jaeger.SpanID(9), jaeger.SpanID(9), true, map[string]string{
				"name": "chenhao",
			})
			sp := opentracing.StartSpan("test_chenhao_1", opentracing.ChildOf(spanContext))
			sp.Finish()

			childSp := opentracing.StartSpan("test_chenhao_2", opentracing.ChildOf(sp.Context()))
			//childSp.SetTag("error", true)
			jaeger.SetServiceErrorTag(childSp.(*jaeger.Span))
			jaeger.BaggageServiceIPs(childSp.(*jaeger.Span), "127.0.0.1")
			jaeger.BaggageSpanTagSelfErr(childSp.(*jaeger.Span))
			childSp.Finish()

			time.Sleep(time.Second)
		}

		time.Sleep(100 * time.Millisecond)
	}
}
