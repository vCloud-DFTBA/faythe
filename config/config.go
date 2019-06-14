package config

import "github.com/spf13/viper"

// GlobalConfig represents all configurations.
type GlobalConfig struct {
	OenStack   OpenStackConfiguration
	StackStorm StackStormConfiguration
	// RestrictedDomain can define an optional regexp pattern to be matched:
	//
	// - {name} matches anything until the next dot.
	//
	// - {name:pattern} matches the given regexp pattern.
	RestrictedDomain string `yaml:"restrictedDomain"`
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
	// Allows all - not restrict any domains
	viper.SetDefault("restrictedDomain", "{domain:.*}")
	var cfg GlobalConfig
	err = viper.Unmarshal(&cfg)
	return err
}
