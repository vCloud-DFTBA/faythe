package config

/*
OpenStackConfiguration stores information needed to authenticate to an OpenStack Cloud.
*/
type OpenStackConfiguration struct {
	// AuthURL specifies the HTTP endpoint that is required to work with
	// the Identity API of the appropriate version. While it's ultimately needed by
	// all of the identity services, it will often be populated by a provider-level
	// function.
	AuthURL    string `yaml:"-"`
	RegionName string `yaml:"regionName,omitempty"`

	// Username is required if using Identity V2 API. Consult with your provider's
	// control panel to discover your account's username. In Identity V3, either
	// UserID or a combination of Username and DomainID or DomainName are needed.
	Username string `yaml:"username,omitempty"`
	UserID   string `yaml:"-"`

	Password string `yaml:"password"`

	// At most one of DomainID and DomainName must be provided if using Username
	// with Identity V3. Otherwise, either are optional.
	DomainName string `yaml:"domainName,omitempty"`
	DomainID   string `yaml:"-"`

	// The ProjectID and ProjectName fields are optional for the Identity V2 API.
	// The same fields are known as project_id and project_name in the Identity
	// V3 API, but are collected as ProjectID and ProjectName here in both cases.
	// Some providers allow you to specify a ProjectName instead of the ProjectId.
	// Some require both. Your provider's authentication policies will determine
	// how these fields influence authentication.
	// If DomainID or DomainName are provided, they will also apply to ProjectName.
	// It is not currently possible to authenticate with Username and a Domain
	// and scope to a Project in a different Domain by using ProjectName. To
	// accomplish that, the ProjectID will need to be provided as the ProjectID
	// option.
	ProjectName string `yaml:"projectId,omitempty"`
	ProjectID   string `yaml:"projectName,omitempty"`

	// UpdateInterval field is the number of seconds that queries the outputs of stacks
	// that was filters with a given listOpts periodically.
	UpdateInterval int `yaml:"updateInterval"`

	// StackTags Lists stacks that contain one or more simple string tags. To specify
	// multiple tags, separate the tags with commas. For example, tag1,tag2
	StackTags string `yaml:"stackTags,omitempty"`
}
