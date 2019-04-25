package auth

import (
	"faythe/utils"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
)

// CreateProvider gets configuration and returns a ProviderClient
func CreateProvider() (*gophercloud.ProviderClient, error) {
	opts := gophercloud.AuthOptions{
		IdentityEndpoint: utils.Getenv("OS_AUTH_URL", viper.GetString("openstack.authURL")),
		Username:         utils.Getenv("OS_USERNAME", viper.GetString("openstack.username")),
		Password:         utils.Getenv("OS_PASSWORD", viper.GetString("openstack.password")),
		DomainName:       utils.Getenv("OS_DOMAIN_NAME", viper.GetString("openstack.domainName")),
		DomainID:         utils.Getenv("OS_DOMAIN_ID", viper.GetString("openstack.domainID")),
		TenantID:         utils.Getenv("OS_TENANT_ID", viper.GetString("openstack.projectID")),
		TenantName:       utils.Getenv("OS_TENANT_NAME", viper.GetString("openstack.projectName")),
	}

	// Create a general client
	provider, err := openstack.AuthenticatedClient(opts)
	if err != nil {
		return nil, errors.Wrap(err, "create OpenStack provider failed")
	}
	return provider, nil
}
