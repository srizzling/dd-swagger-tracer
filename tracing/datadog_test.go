package tracing

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/mocktracer"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

func TestConnection(t *testing.T) {
	assert := assert.New(t)

	// Nothing is listening here.. so error out
	err := verifyConnection("somerandomaddress:0999")
	assert.Error(err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	httpmock.RegisterResponder("GET", "http://localhost:8126/debug/vars", httpmock.NewStringResponder(200, `OK`))

	// Shouldn't error since, an agent exists, and the status is okay
	err = verifyConnection("localhost:8126")
	assert.NoError(err)
}

func TestConnectionNon200StatusCode(t *testing.T) {
	assert := assert.New(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Should error since the agent has responded with a non-200 statuscode
	httpmock.RegisterResponder("GET", "http://localhost:8126/debug/vars", httpmock.NewStringResponder(500, `NOT_OK`))
	err := verifyConnection("localhost:8126")
	assert.Error(err)
}

func TestDiscovery(t *testing.T) {
	assert := assert.New(t)
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	// Should return default localhost because its running in ec2
	err := errors.New("Timeout")
	httpmock.RegisterResponder(
		"GET", "http://169.254.169.254/latest/meta-data/local-ipv4",
		httpmock.NewErrorResponder(err),
	)
	host := discover("")
	assert.Equal("127.0.0.1", host)

	// Should return the private ip from the metadata service.. mocked here
	httpmock.RegisterResponder(
		"GET", "http://169.254.169.254/latest/meta-data/local-ipv4",
		httpmock.NewStringResponder(200, `some_address_from_metadata_service`),
	)

	// Should return default localhost because its running in ec2
	host = discover("")
	assert.Equal("some_address_from_metadata_service", host)
}

func TestConfigure(t *testing.T) {
	assert := assert.New(t)

	config := DDStartTracerConfig{
		Host:        "localhost",
		Port:        8126,
		DebugMode:   true,
		ServiceName: "skeleton",
		SpanTags:    []string{"env:test", "app:skeleton"},
	}

	agentAddr, startOptions := configure(&config)

	assert.Equal(agentAddr, "localhost:8126")

	// Its too hard to actually check the values.. so I'm just going to verify
	// that the length is correct
	// Start Options should only have the following:
	// AgentAddr, DebugMode, and 2 GlobalSpan Tags.
	// The serviceName is actually passed to the handler, and shouldn't exist here
	assert.Len(startOptions, 4)
}

func TestPropogationSpan(t *testing.T) {
	assert := assert.New(t)
	ServiceName = "datadog_test_propogation"

	mt := mocktracer.Start()
	defer mt.Stop()

	pspan := tracer.StartSpan("test")
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	err := tracer.Inject(pspan.Context(), tracer.HTTPHeadersCarrier(r.Header))

	assert.Nil(err)

	// test server
	handler := http.HandlerFunc(
		func(rw http.ResponseWriter, req *http.Request) {
			span, ok := tracer.SpanFromContext(req.Context())
			assert.True(ok)
			assert.Equal(span.(mocktracer.Span).ParentID(), pspan.(mocktracer.Span).SpanID())
		})

	ddMiddleware := Handler(handler)
	ddMiddleware.ServeHTTP(w, r)

}

func TestDatadogSpanTags(t *testing.T) {
	assert := assert.New(t)
	ServiceName = "datadog_test"

	// Start a mocktracer, since we shouldn't need to hit DataDog
	mt := mocktracer.Start()
	defer mt.Stop()

	// Set a global span to test and verify the tag (need it here since statuscode will be checked after the request is completed)
	var span tracer.Span
	resourceName := "/test_resource"

	// test server
	fn := func(w http.ResponseWriter, r *http.Request) {
		var ok bool
		span, ok = tracer.SpanFromContext(r.Context())

		// assert that the span is okay
		assert.True(ok)

		// assert all the custom tags are okay
		assert.Equal(span.(mocktracer.Span).Tag(ext.ServiceName), "datadog_test")
		assert.Equal(span.(mocktracer.Span).Tag(ext.ResourceName), resourceName)
		assert.Equal(span.(mocktracer.Span).Tag(ext.SpanType), "web")
		assert.Equal(span.(mocktracer.Span).Tag(ext.HTTPMethod), "GET")

		// Set a status code here
		w.WriteHeader(http.StatusInternalServerError)
	}

	ts := httptest.NewServer(Handler(http.HandlerFunc(fn)))
	defer ts.Close()

	// test
	res, _ := http.Get(ts.URL + resourceName)

	// Check the span for the StatusCode and verify thats okay too
	assert.Equal(span.(mocktracer.Span).Tag(ext.HTTPCode), strconv.Itoa(res.StatusCode))
}
