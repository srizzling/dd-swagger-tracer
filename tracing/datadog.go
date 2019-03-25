package tracing

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env"
	"github.com/sirupsen/logrus"
	"github.com/srizzling/dd-trace-swagger/pkg/ecs"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/aws/aws-sdk-go/aws/session"

	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go/aws"
)

// ServiceName is a global variable that gets updated when Start() is called, the default is just incase something fails during start up
var ServiceName = "default_service"

const (
	// This Datadog debug path is for verifying a valid connection can be made to the DataDog APM agent,
	// since the agent exposes no API to verify a heartbeat. (/debug/vars is just Datadog apm agent exposesing expvars see: https://golang.org/pkg/expvar/)
	datadogDebugPath = "/debug/vars"

	// Since we're hitting the virtualhost, this should be very quick
	// Another reason to keep this quick is because it _will_ timeout when running locally
	timeoutSeconds = 1

	// A variable pointing to localhost
	localhost = "127.0.0.1"
)

// WrapAWSSession will inject pre and post DataDog spans into a configured session
func WrapAWSSession(s *session.Session, opts ...awstrace.Option) *session.Session {
	return awstrace.WrapSession(s, opts...)
}

func configure(config *DDStartTracerConfig) (string, []tracer.StartOption) {

	// discover host agent host
	agentHost := discover(config.Host)

	agentAddr := fmt.Sprintf("%s:%d", agentHost, config.Port)
	startOpts := []tracer.StartOption{
		tracer.WithAgentAddr(agentAddr),
		tracer.WithDebugMode(config.DebugMode),
	}

	logrus.
		WithField("Config", config).
		Infof("configuring datadog agent")

	// Attach Global Span Tags sourced from configuration (similar to java's implementation)
	// In production this is added by nbos-cloud-ms
	for _, tag := range config.SpanTags {
		s := strings.Split(tag, ":")
		if len(s) == 2 {
			startOpts = append(startOpts, tracer.WithGlobalTag(s[0], s[1]))
		}
	}

	return agentAddr, startOpts
}

// Attempt to discover DataDog host
// if its on ECS grab the EC2 private IP, if not use localhost
func discover(host string) string {
	if host == "" {
		ip, err := ecs.GetPrivateHostIP()
		if err != nil {
			host = localhost
		} else {
			host = ip
		}
	}
	return host
}

func loadEnv() (*DDStartTracerConfig, error) {
	config := &DDStartTracerConfig{}
	err := env.Parse(config)
	if err != nil {
		return config, err
	}
	return config, nil
}

func StartFromEnv() error {
	config, err := loadEnv()
	if err != nil {
		return err
	}
	return Start(config)
}

func Start(config *DDStartTracerConfig) error {
	// Set global variable which is used in the handler
	if config.ServiceName != "" {
		ServiceName = config.ServiceName
	}

	// Generate start options to configure the datadog tracer agent
	agentAddr, startOpts := configure(config)

	// Verify an active agent is available,
	// its up to the calling process to handle this error
	err := verifyConnection(agentAddr)
	if err != nil {
		return err
	}

	// If it gets here, a tracer should be configured to send data to datadog
	tracer.Start(startOpts...)

	return nil
}

func Stop() {
	tracer.Stop()
}

// This verifies that a connection can be made to a Datadog agent,
// if it can't a tracer shouldn't even be started, and its up to the parent to handle the error
func verifyConnection(datadogHost string) error {

	timeout := time.Duration(timeoutSeconds * time.Second)

	datadogDebugURL := fmt.Sprintf("http://%s%s", datadogHost, datadogDebugPath)

	// Call should be done pretty quick since, if its not there it shouldn't stop the application from just spinning
	client := http.Client{
		Timeout: timeout,
	}

	// If its unreachable, this should cry out
	resp, err := client.Get(datadogDebugURL)
	if err != nil {
		return err
	}

	// if the status isn't 200 probably should return an error here..
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("DataDog agent (%s) responded with non-200 status code (expected 200, got %d)", datadogDebugURL, resp.StatusCode)
	}

	return nil
}

// Handler builds a new middleware handler
func Handler(next http.Handler) http.Handler {

	globalOps := []ddtrace.StartSpanOption{
		tracer.ServiceName(ServiceName),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// request specific configuration
		opts := append(globalOps, []ddtrace.StartSpanOption{
			tracer.ResourceName(r.URL.RequestURI()),
			tracer.SpanType(ext.SpanTypeWeb),
			tracer.Tag(ext.HTTPMethod, r.Method),
			tracer.Tag(ext.HTTPURL, r.URL),
		}...)

		// check context for spanids, if exists associate current span as a child.
		if spanctx, err := tracer.Extract(tracer.HTTPHeadersCarrier(r.Header)); err == nil {
			opts = append(opts, tracer.ChildOf(spanctx))
		}

		// create a datadog trace span with context
		span, ctx := tracer.StartSpanFromContext(r.Context(), "http.request", opts...)

		// this records the status details
		wws := NewResponseWriterWithStatus(w)

		// ... and defer writing it until the request is done
		defer func() {
			span.SetTag(ext.HTTPCode, strconv.Itoa(wws.statusCode))
			span.Finish()
		}()

		// continue serving request
		next.ServeHTTP(wws, r.WithContext(ctx))

	})
}

// Since ResponseWrite is a write-only struct, we have to wrap it to be able to store the status code for logging
// Idea borrowed from here:
// https://gist.github.com/ciaranarcher/abccf50cb37645ca27fa
type responseWriterWithStatus struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriterWithStatus(w http.ResponseWriter) *responseWriterWithStatus {
	// WriteHeader(int) is not called if our response implicitly returns 200 OK, so
	// we default to that status code.
	return &responseWriterWithStatus{w, http.StatusOK}
}

func (lrw *responseWriterWithStatus) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
