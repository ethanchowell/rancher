//go:build (validation || sanity) && !infra.any && !infra.aks && !infra.eks && !infra.rke2k3s && !infra.gke && !infra.rke1 && !cluster.any && !cluster.custom && !cluster.nodedriver && !extended && !stress

package rke1

import (
	"testing"

	"github.com/rancher/rancher/tests/framework/clients/rancher"
	management "github.com/rancher/rancher/tests/framework/clients/rancher/generated/management/v3"
	"github.com/rancher/rancher/tests/framework/extensions/clusters"
	"github.com/rancher/rancher/tests/framework/extensions/clusters/kubernetesversions"
	"github.com/rancher/rancher/tests/framework/extensions/provisioninginput"
	"github.com/rancher/rancher/tests/framework/extensions/users"
	password "github.com/rancher/rancher/tests/framework/extensions/users/passwordgenerator"
	"github.com/rancher/rancher/tests/framework/pkg/config"
	"github.com/rancher/rancher/tests/framework/pkg/environmentflag"
	namegen "github.com/rancher/rancher/tests/framework/pkg/namegenerator"
	"github.com/rancher/rancher/tests/framework/pkg/session"
	"github.com/rancher/rancher/tests/v2/validation/provisioning/permutations"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CustomClusterProvisioningTestSuite struct {
	suite.Suite
	client             *rancher.Client
	session            *session.Session
	standardUserClient *rancher.Client
	provisioningConfig *provisioninginput.Config
}

func (c *CustomClusterProvisioningTestSuite) TearDownSuite() {
	c.session.Cleanup()
}

func (c *CustomClusterProvisioningTestSuite) SetupSuite() {
	testSession := session.NewSession()
	c.session = testSession

	c.provisioningConfig = new(provisioninginput.Config)
	config.LoadConfig(provisioninginput.ConfigurationFileKey, c.provisioningConfig)

	client, err := rancher.NewClient("", testSession)
	require.NoError(c.T(), err)

	c.client = client

	c.provisioningConfig.RKE1KubernetesVersions, err = kubernetesversions.Default(c.client, clusters.RKE1ClusterType.String(), c.provisioningConfig.RKE1KubernetesVersions)
	require.NoError(c.T(), err)

	enabled := true
	var testuser = namegen.AppendRandomString("testuser-")
	var testpassword = password.GenerateUserPassword("testpass-")
	user := &management.User{
		Username: testuser,
		Password: testpassword,
		Name:     testuser,
		Enabled:  &enabled,
	}

	newUser, err := users.CreateUserWithRole(client, user, "user")
	require.NoError(c.T(), err)

	newUser.Password = user.Password

	standardUserClient, err := client.AsUser(newUser)
	require.NoError(c.T(), err)

	c.standardUserClient = standardUserClient
}

func (c *CustomClusterProvisioningTestSuite) TestProvisioningRKE1CustomCluster() {
	nodeRolesAll := []provisioninginput.NodePools{provisioninginput.AllRolesNodePool}
	nodeRolesShared := []provisioninginput.NodePools{provisioninginput.EtcdControlPlaneNodePool, provisioninginput.WorkerNodePool}
	nodeRolesDedicated := []provisioninginput.NodePools{provisioninginput.EtcdNodePool, provisioninginput.ControlPlaneNodePool, provisioninginput.WorkerNodePool}

	require.GreaterOrEqual(c.T(), len(c.provisioningConfig.CNIs), 1)

	tests := []struct {
		name      string
		nodePools []provisioninginput.NodePools
		client    *rancher.Client
		runFlag   bool
	}{
		{"1 Node all roles " + provisioninginput.AdminClientName.String(), nodeRolesAll, c.client, c.client.Flags.GetValue(environmentflag.Long)},
		{"1 Node all roles " + provisioninginput.StandardClientName.String(), nodeRolesAll, c.standardUserClient, c.client.Flags.GetValue(environmentflag.Short) || c.client.Flags.GetValue(environmentflag.Long)},
		{"2 nodes - etcd/cp roles per 1 node " + provisioninginput.AdminClientName.String(), nodeRolesShared, c.client, c.client.Flags.GetValue(environmentflag.Long)},
		{"2 nodes - etcd/cp roles per 1 node " + provisioninginput.StandardClientName.String(), nodeRolesShared, c.standardUserClient, c.client.Flags.GetValue(environmentflag.Short) || c.client.Flags.GetValue(environmentflag.Long)},
		{"3 nodes - 1 role per node " + provisioninginput.AdminClientName.String(), nodeRolesDedicated, c.client, c.client.Flags.GetValue(environmentflag.Long)},
		{"3 nodes - 1 role per node " + provisioninginput.StandardClientName.String(), nodeRolesDedicated, c.standardUserClient, c.client.Flags.GetValue(environmentflag.Long)},
	}
	for _, tt := range tests {
		if !tt.runFlag {
			c.T().Logf("SKIPPED")
			continue
		}

		provisioningConfig := *c.provisioningConfig
		provisioningConfig.NodePools = tt.nodePools
		permutations.RunTestPermutations(&c.Suite, tt.name, tt.client, &provisioningConfig, permutations.RKE1CustomCluster, nil, nil)
	}
}

func (c *CustomClusterProvisioningTestSuite) TestProvisioningRKE1CustomClusterDynamicInput() {
	require.GreaterOrEqual(c.T(), len(c.provisioningConfig.CNIs), 1)

	if len(c.provisioningConfig.NodePools) == 0 {
		c.T().Skip()
	}

	tests := []struct {
		name   string
		client *rancher.Client
	}{
		{provisioninginput.AdminClientName.String(), c.client},
		{provisioninginput.StandardClientName.String(), c.standardUserClient},
	}
	for _, tt := range tests {
		permutations.RunTestPermutations(&c.Suite, tt.name, tt.client, c.provisioningConfig, permutations.RKE1CustomCluster, nil, nil)
	}
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestCustomClusterRKE1ProvisioningTestSuite(t *testing.T) {
	suite.Run(t, new(CustomClusterProvisioningTestSuite))
}
