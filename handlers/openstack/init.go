package openstack

import (
	"net/url"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"

	"github.com/ntk148v/faythe/config"
	"github.com/ntk148v/faythe/utils"
)

// ServiceName represents the name of service, for example: heat, nova
type ServiceName string

// ServicePort represents the port of service, for example: 8004 - heat.
type ServicePort string

// ServiceVersion represents the current version of service, for example: v1
type ServiceVersion string

var (
	logger         *utils.Flogger
	once           sync.Once
	existingAlerts utils.SharedValue
	authToken      tokens.Token
)

// Service represents OpenStack service with name, version and port.
func generateEndpoints(name ServiceName, port ServicePort, version ServiceVersion) {
	confs := config.Get()
	for _, v := range confs.OpenStackConfigs {
		if v.Endpoints == nil {
			v.Endpoints = make(map[string]string)
		}
		if _, ok := v.Endpoints[string(name)]; !ok {
			authURL, _ := url.Parse(v.AuthURL)
			svcURL := url.URL{
				Scheme: authURL.Scheme,
				Host:   strings.Replace(authURL.Host, authURL.Port(), string(port), -1),
				Path:   string(version),
			}
			v.Endpoints[string(name)] = svcURL.String()
		}
	}
	config.Set(confs)
}
