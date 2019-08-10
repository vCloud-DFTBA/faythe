package openstack

import (
	"net/url"
	"strings"
	"sync"

	"github.com/gophercloud/gophercloud/openstack/identity/v3/tokens"

	"faythe/config"
	"faythe/utils"
)

// service represents OpenStack service with name, version and port.
type service struct {
	name    string
	version string
	port    string
}

var (
	logger         *utils.Flogger
	once           sync.Once
	existingAlerts utils.SharedValue
	authToken      tokens.Token
)

// Service represents OpenStack service with name, version and port.
func generateEndpoints(svc service) {
	confs := config.Get()
	for _, v := range confs.OpenStackConfigs {
		if v.Endpoints == nil {
			v.Endpoints = make(map[string]string)
		}
		if _, ok := v.Endpoints[svc.name]; !ok {
			authURL, _ := url.Parse(v.AuthURL)
			svcURL := url.URL{
				Scheme: authURL.Scheme,
				Host:   strings.Replace(authURL.Host, authURL.Port(), svc.port, -1),
				Path:   svc.version,
			}
			v.Endpoints[svc.name] = svcURL.String()
		}
	}
	config.Set(confs)
}
