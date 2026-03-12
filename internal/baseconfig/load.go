package baseconfig

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
)

// LoadYAMLInto reads a YAML file and unmarshals it into `out`.
// `out` must be a pointer to a struct (or map) defined by the caller.
func loadYAMLInto(path string, out any) error {
	if path == "" {
		return fmt.Errorf("config path is empty")
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %q: %w", path, err)
	}

	if err := yaml.Unmarshal(b, out); err != nil {
		return fmt.Errorf("unmarshal yaml %q: %w", path, err)
	}

	return nil
}

func LoadConfig(configFilename string, cfg any) error {
	configFile := configFilename
	configDir := os.Getenv("CONFIG_FILE")
	if configDir != "" {
		configFile = configDir
	}

	err := loadYAMLInto(configFile, cfg)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	return nil
}
