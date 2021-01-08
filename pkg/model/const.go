// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

const (
	// OpenStackType represents a OpenStack type
	OpenStackType string = "openstack"

	// PrometheusType is a Prometheus backend
	PrometheusType = "prometheus"

	FixedDelay   = "fixed"
	BackoffDelay = "backoff"

	// DefaultCloudPrefix is the default etcd prefix for Cloud data
	DefaultCloudPrefix = "/clouds"

	DefaultHealerPrefix   = "/healers"
	DefaultHealerQuery    = "up{job=~\".*compute-cadvisor.*|.*compute-node.*\"} < 1"
	DefaultHealerInterval = "18s"
	DefaultHealerDuration = "3m"

	DefaultNResolverPrefix   = "/nresolvers"
	DefaultNResolverQuery    = "node_uname_info"
	DefaultNResolverInterval = "30s"

	// DefaultScalerPrefix is the etcd default prefix for scaler
	DefaultScalerPrefix = "/scalers"

	// DefaultSilencePrefix is default etcd prefix for Silences
	DefaultSilencePrefix             = "/silences"
	DefaultSilenceValidationInterval = "30s"
	DefaultSyncSilencesInterval      = "20s"

	// DefaultUserPrefix is default etcd prefix for Users
	DefaultUsersPrefix = "/users"

	// DefaultPoliciesPrefix is default etcd prefix for policies
	// This prefix is a bit different than others, it doesn't start with a slash '/'
	DefaultPoliciesPrefix = "policies"
)

const DefaultMaxNumberOfInstances int = 3
