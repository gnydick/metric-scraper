package config

import (
    "encoding/json"
    "log"
    "os"
)

type Config struct {
    debug        bool
    kind         string
    disco        string
    ident        string
    deploymentId string
    interval     string // duration syntax
    orch         string
    metric       string
    sink         string
    mode         string // deployed or development - running in a cluster vs running outside cluster
    optionals    map[string]map[string]string
}

func (c Config) init() {
    c.optionals = make(map[string]map[string]string)
    c.optionals["deployed"] = make(map[string]string)
    c.optionals["development"] = make(map[string]string)
}

func (c *Config) Optionals() *map[string]map[string]string {
    return &c.optionals
}

func (c *Config) Mode() string {
    return c.mode
}

func (c *Config) Metric() string {
    return c.metric
}

func (c *Config) Orch() string {
    return c.orch
}

func (c *Config) Interval() string {
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

func (c *Config) Kind() string {
    return c.kind
}

func (c *Config) Debug() bool {
    return c.debug
}
func (c *Config) Sink() string {
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
        c.kind = kind
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
        c.interval = interval
    }
    sink := os.Getenv("SINK")
    if len(sink) == 0 {
        log.Fatal("Must specify sink, SINK env var.")

    } else {
        c.sink = sink
    }

    mode := os.Getenv("MODE")
    if len(sink) == 0 {
        log.Fatal("Must specify sink, SINK env var.")

    } else {
        c.mode = mode
    }

    kubeConfig := os.Getenv("KUBE_CONFIG")
    if len(kubeConfig) > 0 {
        c.optionals["development"]["kubeConfig"] = kubeConfig
    }

    return
}

func FileBuild(configFile string) (Config) {
    configuration := Config{}
    data := make(map[string]interface{})
    file, _err := os.Open(configFile)
    if _err != nil {
        log.Fatal(_err.Error())
    }
    decoder := json.NewDecoder(file)
    _err = decoder.Decode(&data)
    if _err != nil {
        log.Fatal(_err.Error())
    }
    configuration.debug = data["debug"].(bool)
    configuration.kind = data["kind"].(string)
    configuration.disco = data["disco"].(string)
    configuration.ident = data["ident"].(string)
    configuration.deploymentId = data["deploymentId"].(string)
    configuration.interval = data["interval"].(string)
    configuration.orch = data["orch"].(string)
    configuration.metric = data["metric"].(string)
    configuration.orch = data["orch"].(string)
    configuration.sink = data["sink"].(string)
    configuration.mode = data["mode"].(string)
    opts := data["optionals"]

    m := make(map[string]map[string]string)
    bites, _err := json.Marshal(&opts)
    if _err != nil {
        log.Fatalf("error: %v", _err)
    }
    _err = json.Unmarshal(bites, &m)
    if _err != nil {
        log.Fatalf("error: %v", _err)
    }
    configuration.optionals = m

    return configuration
}
