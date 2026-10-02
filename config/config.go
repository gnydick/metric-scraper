package config

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gnydick/metric-scraper/telemetry"
)

// Kind is what the scraper scrapes.
type Kind string

const (
	KindCadvisor Kind = "cadvisor"
	KindService  Kind = "service"
)

// SinkKind is where the scraper sends metrics.
type SinkKind string

const (
	SinkOpentsdb SinkKind = "opentsdb"
)

// Mode is where the scraper runs: in a cluster, or outside one.
type Mode string

const (
	ModeDeployed    Mode = "deployed"
	ModeDevelopment Mode = "development"
)

// Config is a validated configuration. The fields are private and FileBuild is the one way to fill
// them, so a Config from FileBuild has a known kind, sink and mode and a positive interval. The
// zero Config has none of them; NewScraper refuses it.
type Config struct {
	debug        bool
	kind         Kind
	disco        string
	ident        string
	deploymentId string
	interval     time.Duration
	orch         string
	metric       string
	sink         SinkKind
	mode         Mode
	optionals    map[string]map[string]string
	// metricsPrefix starts every name on the scraper's own metrics page.
	metricsPrefix string
}

func (c *Config) MetricsPrefix() string {
	return c.metricsPrefix
}

func (c Config) init() {
	c.optionals = make(map[string]map[string]string)
	c.optionals["deployed"] = make(map[string]string)
	c.optionals["development"] = make(map[string]string)
}

func (c *Config) Optionals() *map[string]map[string]string {
	return &c.optionals
}

func (c *Config) Mode() Mode {
	return c.mode
}

func (c *Config) Metric() string {
	return c.metric
}

func (c *Config) Orch() string {
	return c.orch
}

func (c *Config) Interval() time.Duration {
	return c.interval
}

func (c *Config) DeploymentId() string {
	return c.deploymentId
}

func (c *Config) Ident() string {
	return c.ident
}

func (c *Config) Disco() string {
	return c.disco
}

func (c *Config) Kind() Kind {
	return c.kind
}

func (c *Config) Debug() bool {
	return c.debug
}
func (c *Config) Sink() SinkKind {
	return c.sink
}

func EnvBuild() (config Config) {
	c := Config{}

	if os.Getenv("DEBUG") == "true" {
		c.debug = true
	}
	deploymentId := os.Getenv("DEPLOYMENT_ID")
	if len(deploymentId) == 0 {
		log.Fatal("Must specify DEPLOYMENT_ID env var.")
	} else {
		c.deploymentId = deploymentId
	}
	kind := os.Getenv("KIND")
	if len(kind) == 0 {
		log.Fatal("Must specify scraper KIND env var.")
	} else {
		c.kind = Kind(kind)
	}

	disco := os.Getenv("DISCO")
	if len(disco) == 0 {
		log.Fatal("Must specify target, DISCO env var.")
	} else {
		c.disco = disco
	}

	orch := os.Getenv("ORCH")
	if len(orch) == 0 {
		log.Fatal("Must specify orch endpoint, ORCH env var.")

	} else {
		c.orch = orch
	}
	interval := os.Getenv("INTERVAL")
	if len(interval) == 0 {
		log.Fatal("Must specify interval, INTERVAL env var.")

	} else {
		c.interval, _ = time.ParseDuration(interval)
	}
	sink := os.Getenv("SINK")
	if len(sink) == 0 {
		log.Fatal("Must specify sink, SINK env var.")

	} else {
		c.sink = SinkKind(sink)
	}

	mode := os.Getenv("MODE")
	if len(sink) == 0 {
		log.Fatal("Must specify sink, SINK env var.")

	} else {
		c.mode = Mode(mode)
	}

	kubeConfig := os.Getenv("KUBE_CONFIG")
	if len(kubeConfig) > 0 {
		c.optionals["development"]["kubeConfig"] = kubeConfig
	}

	return
}

// FileBuild loads and validates the config file. A file that cannot be read, is not JSON, lacks a
// field, has a field of the wrong type or holds a value the scraper does not know gives an error
// naming the file and the field, and no usable Config.
func FileBuild(configFile string) (Config, error) {
	file, err := os.Open(configFile)
	if err != nil {
		return Config{}, fmt.Errorf("config file %s: %w", configFile, err)
	}
	defer file.Close()

	data := make(map[string]interface{})
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		return Config{}, fmt.Errorf("config file %s: not valid JSON: %w", configFile, err)
	}

	configuration, err := validate(data)
	if err != nil {
		return Config{}, fmt.Errorf("config file %s: %w", configFile, err)
	}
	return configuration, nil
}

// validate turns decoded config data into a Config, or says which field is wrong.
func validate(data map[string]interface{}) (Config, error) {
	var configuration Config
	var err error

	debug, present := data["debug"]
	if !present {
		return Config{}, fmt.Errorf("field %q is missing", "debug")
	}
	isBool := false
	if configuration.debug, isBool = debug.(bool); !isBool {
		return Config{}, fmt.Errorf("field %q must be true or false", "debug")
	}

	text := func(name string) string {
		if err != nil {
			return ""
		}
		value, present := data[name]
		if !present {
			err = fmt.Errorf("field %q is missing", name)
			return ""
		}
		s, isString := value.(string)
		if !isString {
			err = fmt.Errorf("field %q must be a string", name)
			return ""
		}
		return s
	}
	kind := text("kind")
	configuration.disco = text("disco")
	configuration.ident = text("ident")
	configuration.deploymentId = text("deploymentId")
	interval := text("interval")
	configuration.orch = text("orch")
	configuration.metric = text("metric")
	sink := text("sink")
	mode := text("mode")
	if err != nil {
		return Config{}, err
	}

	switch Kind(kind) {
	case KindCadvisor, KindService:
		configuration.kind = Kind(kind)
	default:
		return Config{}, fmt.Errorf("field %q: unknown value %q, want %q or %q", "kind", kind, KindCadvisor, KindService)
	}
	switch SinkKind(sink) {
	case SinkOpentsdb:
		configuration.sink = SinkKind(sink)
	default:
		return Config{}, fmt.Errorf("field %q: unknown value %q, want %q", "sink", sink, SinkOpentsdb)
	}
	switch Mode(mode) {
	case ModeDeployed, ModeDevelopment:
		configuration.mode = Mode(mode)
	default:
		return Config{}, fmt.Errorf("field %q: unknown value %q, want %q or %q", "mode", mode, ModeDeployed, ModeDevelopment)
	}

	// A field may be empty only when nothing reads it. The service kind looks its target up by
	// disco, and the opentsdb sink looks OpenTSDB up by metric.
	if configuration.kind == KindService && configuration.disco == "" {
		return Config{}, fmt.Errorf("field %q must not be empty when kind is %q", "disco", KindService)
	}
	if configuration.sink == SinkOpentsdb && configuration.metric == "" {
		return Config{}, fmt.Errorf("field %q must not be empty when sink is %q", "metric", SinkOpentsdb)
	}

	configuration.interval, err = time.ParseDuration(interval)
	if err != nil {
		return Config{}, fmt.Errorf("field %q: %q is not a duration such as \"30s\"", "interval", interval)
	}
	if configuration.interval <= 0 {
		return Config{}, fmt.Errorf("field %q: %q must be more than zero", "interval", interval)
	}

	// optionals may be left out. When present it is a map of maps of strings.
	optionals := make(map[string]map[string]string)
	bites, err := json.Marshal(data["optionals"])
	if err == nil {
		err = json.Unmarshal(bites, &optionals)
	}
	if err != nil {
		return Config{}, fmt.Errorf("field %q must be a map of maps of strings", "optionals")
	}
	configuration.optionals = optionals

	// metricsPrefix may be left out. When present it must be able to start a metric name (#53).
	configuration.metricsPrefix = telemetry.DefaultPrefix
	if value, present := data["metricsPrefix"]; present {
		prefix, isString := value.(string)
		if !isString {
			return Config{}, fmt.Errorf("field %q must be a string", "metricsPrefix")
		}
		if err := telemetry.CheckPrefix(prefix); err != nil {
			return Config{}, fmt.Errorf("field %q: %w", "metricsPrefix", err)
		}
		configuration.metricsPrefix = prefix
	}

	return configuration, nil
}
