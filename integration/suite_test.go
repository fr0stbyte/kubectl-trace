package integration

import (
	"crypto/rand"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-check/check"
	"github.com/iovisor/kubectl-trace/pkg/version"
	"gotest.tools/icmd"
	"sigs.k8s.io/kind/pkg/cluster"
	"sigs.k8s.io/kind/pkg/cluster/config/encoding"
)

var (
	KubectlTraceBinary = os.Getenv("TEST_KUBECTLTRACE_BINARY")
	KindImageTag       = os.Getenv("TEST_KIND_IMAGETAG")
)

type KubectlTraceSuite struct {
	kubeConfigPath string
	kindContext    *cluster.Context
}

func init() {
	if KubectlTraceBinary == "" {
		KubectlTraceBinary = "kubectl-trace"
	}

	if KindImageTag == "" {
		KindImageTag = "kindest/node:v1.12.3"
	}
	check.Suite(&KubectlTraceSuite{})
}

func (k *KubectlTraceSuite) SetUpSuite(c *check.C) {
	cfg, err := encoding.Load("")
	c.Assert(err, check.IsNil)
	retain := false
	wait := time.Duration(0)

	err = cfg.Validate()
	c.Assert(err, check.IsNil)

	clusterName, err := generateClusterName()
	c.Assert(err, check.IsNil)
	kctx := cluster.NewContext(clusterName)

	err = kctx.Create(cfg, retain, wait)
	c.Assert(err, check.IsNil)
	k.kindContext = kctx

	nodes, err := kctx.ListNodes()

	c.Assert(err, check.IsNil)

	// copy the bpftrace image to the nodes
	for _, n := range nodes {
		loadcomm := fmt.Sprintf("docker save %s | docker exec -i %s docker load", version.ImageNameTag(), n.String())
		res := icmd.RunCommand("bash", "-c", loadcomm)
		c.Assert(res.Error, check.IsNil)
	}
}

func (s *KubectlTraceSuite) TearDownSuite(c *check.C) {
	err := s.kindContext.Delete()
	c.Assert(err, check.IsNil)
}

func Test(t *testing.T) { check.TestingT(t) }

func (k *KubectlTraceSuite) KubectlTraceCmd(c *check.C, args ...string) string {
	args = append([]string{fmt.Sprintf("--kubeconfig=%s", k.kindContext.KubeConfigPath())}, args...)
	res := icmd.RunCommand(KubectlTraceBinary, args...)
	c.Assert(res.ExitCode, check.Equals, icmd.Success.ExitCode)
	return res.Combined()
}

func generateClusterName() (string, error) {
	buf := make([]byte, 10)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return strings.ToLower(fmt.Sprintf("%X", buf)), nil
}
