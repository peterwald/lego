package acme

import (
	"fmt"
	"os"
	"testing"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/dns/v1"

	"github.com/stretchr/testify/assert"
)

var (
	gcloudLiveTest bool
	gcloudProject  string
	gcloudDomain   string
)

func init() {
	gcloudProject = os.Getenv("GCE_PROJECT")
	gcloudDomain = os.Getenv("GCE_DOMAIN")
	fmt.Printf("gcloud vars: %q %q\n", gcloudProject, gcloudDomain)
	if _, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope); err == nil && len(gcloudProject) > 0 && len(gcloudDomain) > 0 {
		// disable live tests if local credentials cannot be loaded.
		fmt.Printf("Enabling live test....\n")
		gcloudLiveTest = true
	}
	fmt.Printf("gcloudLiveTest: %v\n", gcloudLiveTest)
}

func restoreGCloudEnv() {
	os.Setenv("GCE_PROJECT", gcloudProject)
}

func TestNewDNSProviderGoogleCloudValid(t *testing.T) {
	os.Setenv("GCE_PROJECT", "")
	_, err := NewDNSProviderGoogleCloud("my-project")
	assert.NoError(t, err)
	restoreGCloudEnv()
}

func TestNewDNSProviderGoogleCloudValidEnv(t *testing.T) {
	os.Setenv("GCE_PROJECT", "my-project")
	_, err := NewDNSProviderGoogleCloud("")
	assert.NoError(t, err)
	restoreGCloudEnv()
}

func TestNewDNSProviderGoogleCloudMissingCredErr(t *testing.T) {
	os.Setenv("GCE_PROJECT", "")
	_, err := NewDNSProviderGoogleCloud("")
	assert.EqualError(t, err, "Google Cloud credentials missing")
	restoreGCloudEnv()
}

func TestLiveGoogleCloudPresent(t *testing.T) {
	t.logf("Live Test: %v", gcloudLiveTest)
	if !gcloudLiveTest {
		t.Skip("skipping live test")
	}

	provider, err := NewDNSProviderGoogleCloud(gcloudProject)
	assert.NoError(t, err)

	err = provider.Present(gcloudDomain, "", "123d==")
	assert.NoError(t, err)
}

func TestLiveGoogleCloudCleanUp(t *testing.T) {
	t.logf("Live Test: %v", gcloudLiveTest)
	if !gcloudLiveTest {
		t.Skip("skipping live test")
	}

	time.Sleep(time.Second * 1)

	provider, err := NewDNSProviderGoogleCloud(gcloudProject)
	assert.NoError(t, err)

	err = provider.CleanUp(gcloudDomain, "", "123d==")
	assert.NoError(t, err)
}
