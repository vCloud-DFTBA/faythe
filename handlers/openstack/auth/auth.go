package auth

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/pkg/errors"

	"faythe/config"
)

// CreateProvider gets configuration and returns a ProviderClient
func CreateProvider(conf config.OpenStackConfig) (*gophercloud.ProviderClient, error) {
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
