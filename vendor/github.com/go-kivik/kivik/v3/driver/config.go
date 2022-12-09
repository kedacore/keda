package driver

import "context"

// Config represents all the config sections.
type Config map[string]ConfigSection

// ConfigSection represents all key/value pairs for a section of configuration.
type ConfigSection map[string]string

// Configer is an optional interface that may be implemented by a Client to
// allow access to reading and setting server configuration.
type Configer interface {
	Config(ctx context.Context, node string) (Config, error)
	ConfigSection(ctx context.Context, node, section string) (ConfigSection, error)
	ConfigValue(ctx context.Context, node, section, key string) (string, error)
	SetConfigValue(ctx context.Context, node, section, key, value string) (string, error)
	DeleteConfigKey(ctx context.Context, node, section, key string) (string, error)
}
