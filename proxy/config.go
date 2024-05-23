package proxy

const (
	defaultPort      = 8080
	defaultLogLevel  = "info"
	defaultLogFormat = "plain"
)

var (
	emptyConfig   = Config{}
	defaultConfig = Config{Port: defaultPort, LogLevel: defaultLogLevel, LogFormat: defaultLogFormat}
)

// Config represents the service configuration
// struct.
type Config struct {
	Port      int    `yaml:"port"`
	LogLevel  string `yaml:"loglevel"`
	LogFormat string `yaml:"logformat"`
	URLs      string `yaml:"urls"` // must be supplied by user
}

// Sanitize will support a lazy user by ensuring that empty config file
// fields are replaced with default values.
func (c *Config) Sanitize() {
	if c.Port == 0 {
		c.Port = defaultPort
	}
	if c.LogLevel == "" {
		c.LogLevel = defaultLogLevel
	}
	if c.LogFormat == "" {
		c.LogFormat = defaultLogFormat
	}
}
