// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package model

const (
	// OpenStackType represents a OpenStack type
	OpenStackType string = "openstack"

	FixedDelay   string = "fixed"
	BackoffDelay string = "backoff"

	// DefaultCloudPrefix is the default etcd prefix for Cloud data
	DefaultCloudPrefix string = "/clouds"

	DefaultHealerPrefix   string = "/healers"
	DefaultHealerQuery    string = "up{job=~\".*compute-cadvisor.*|.*compute-node.*\"} < 1"
	DefaultHealerInterval string = "1m"
	DefaultHealerDuration string = "3m"

	DefaultNResolverPrefix   string = "/nresolvers"
	DefaultNResolverQuery    string = "node_uname_info"
	DefaultNResolverInterval string = "60s"

	// DefaultScalerPrefix is the etcd default prefix for scaler
	DefaultScalerPrefix string = "/scalers"
)
