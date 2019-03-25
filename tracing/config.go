package tracing

// DDStartTracerConfig is an
type DDStartTracerConfig struct {
	Host        string   `env:"DATADOG_HOST"`
	Port        int      `env:"DATADOG_APM_PORT" envDefault:"8126"`
	DebugMode   bool     `env:"DD_APM_DEBUG" envDefault:"false"`
	ServiceName string   `env:"DD_SERVICE_NAME"`
	SpanTags    []string `env:"DD_TRACE_SPAN_TAGS" envSeparator:","`
}

func (c *DDStartTracerConfig) WithHost(host string) *DDStartTracerConfig {
	c.Host = host
	return c
}

func (c *DDStartTracerConfig) WithPort(port int) *DDStartTracerConfig {
	c.Port = port
	return c
}

func (c *DDStartTracerConfig) WithDebugMode(debugMode bool) *DDStartTracerConfig {
	c.DebugMode = debugMode
	return c
}

func (c *DDStartTracerConfig) WithServiceName(serviceName string) *DDStartTracerConfig {
	c.ServiceName = serviceName
	return c
}

func (c *DDStartTracerConfig) WithTags(tags []string) *DDStartTracerConfig {
	c.SpanTags = tags
	return c
}
