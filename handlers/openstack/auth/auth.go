package auth

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/pkg/errors"

	"github.com/ntk148v/faythe/config"
)

// CreateProvider gets configuration and returns a ProviderClient
func CreateProvider(conf *config.OpenStackConfig) (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: conf.AuthURL,
		Username:         conf.Username,
		Password:         string(conf.Password),
		DomainName:       conf.DomainName,
		DomainID:         conf.DomainID,
		TenantID:         conf.ProjectID,
		TenantName:       conf.ProjectName,
	}

	// Create a general client
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, errors.Wrap(err, "create OpenStack provider failed")
	}
	return provider, nil
}

// CreateClient creates a ServiceClient that may be used to access the v3
// identity service.
func CreateClient(opsConf *config.OpenStackConfig) (*gophercloud.ServiceClient, error) {
	provider, err := CreateProvider(opsConf)
	if err != nil {
		return nil, err
	}
	return openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{Region: opsConf.RegionName})
}
