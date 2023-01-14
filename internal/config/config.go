package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SSH struct {
	Addr     string   `yaml:"addr"`
	HostKeys []string `yaml:"host_keys"`
}

type User struct {
	Role       string   `yaml:"role"`
	SHA256Keys []string `yaml:"sha256_keys"`
}

type DB struct {
	StorePath string `yaml:"store_path"`
}

type Config struct {
	Debug             bool            `yaml:"debug"`
	SSH               SSH             `yaml:"ssh"`
	DB                DB              `yaml:"db"`
	AllowRegistration bool            `yaml:"allow_registration"`
	Users             map[string]User `yaml:"users"`
}

func LoadConfig(configs ...string) (*Config, error) {
	out := &Config{
		Debug: true,
		SSH: SSH{
			Addr: ":2022",
			HostKeys: []string{
				"ssh_host_rsa_key",
				"ssh_host_ecdsa_key",
				"ssh_host_ed25519_key",
			},
		},
		DB: DB{
			StorePath: "./db",
		},
	}

	if len(configs) == 0 {
		return out, nil
	}

	for _, cfgPath := range configs {
		err := func() error {
			f, err := os.Open(cfgPath)
			if err != nil {
				return fmt.Errorf("unable to open config file: %w", err)
			}
			defer func() { _ = f.Close() }()

			if err := yaml.NewDecoder(f).Decode(&out); err != nil {
				return fmt.Errorf("invalid config: %w", err)
			}

			return nil
		}()
		if err != nil {
			return nil, fmt.Errorf("unable to load config %q: %w", cfgPath, err)
		}
	}

	return out, nil
}
