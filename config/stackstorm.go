package config

/*
StackStormConfiguration stores information needed to forward
request to an StackStorm instance.
*/
type StackStormConfiguration struct {
	Host   string `yaml:"host"`
	Rule   string `yaml:"rule"`
	APIKey string `yaml:"apiKey"`
}
