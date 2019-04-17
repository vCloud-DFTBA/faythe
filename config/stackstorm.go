package config

import "faythe/utils"

/*
StackStormConfiguration stores information needed to forward
request to an StackStorm instance.
*/
type StackStormConfiguration struct {
	Host   string       `yaml:"host"`
	APIKey utils.Secret `yaml:"apiKey"`
}
