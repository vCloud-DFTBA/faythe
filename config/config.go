package config

import "github.com/spf13/viper"

type GlobalConfig struct {
	OpenStack  OpenStackConfiguration
	StackStorm StackStormConfiguration
}

func load(cp string) {
	viper.SetConfigName("config")
	viper.AddConfigPath(cp)
}
