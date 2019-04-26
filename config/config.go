package config

import "github.com/spf13/viper"

// GlobalConfig represents all configurations.
type GlobalConfig struct {
	OpenStack  OpenStackConfiguration
	StackStorm StackStormConfiguration
}

// Load generates a configuration instance which will be passed around the codebase.
func Load(cp string) error {
	viper.SetConfigName("config")
	viper.AddConfigPath(cp)

	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	// Set default values
	viper.SetDefault("openstack.updateInterval", 30)
	var cfg GlobalConfig
	err = viper.Unmarshal(&cfg)
	return err
}
