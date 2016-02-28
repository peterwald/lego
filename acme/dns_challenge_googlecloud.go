package acme

import (
	"fmt"
	"os"
	"strings"
	"time"

	"golang.org/x/net/context"
	"golang.org/x/oauth2/google"

	"google.golang.org/api/dns/v1"
)

// DNSProviderGoogleCloud is an implementation of the DNSProvider interface.
type DNSProviderGoogleCloud struct {
	project string
	client  *dns.Service
}

// NewDNSProviderGoogleCloud returns a DNSProviderGoogleCloud instance with a configured gcloud client.
// Authentication is done using the local account credentials managed by the gcloud utility.
func NewDNSProviderGoogleCloud(project string) (*DNSProviderGoogleCloud, error) {
	if project == "" {
		project = gcloudEnvAuth()
	}
	if project == "" {
		return nil, fmt.Errorf("Google Cloud credentials missing")
	}

	client, err := google.DefaultClient(context.Background(), dns.NdevClouddnsReadwriteScope)
	if err != nil {
		return nil, fmt.Errorf("Unable to get Google Cloud client: %v", err)
	}
	svc, err := dns.New(client)
	if err != nil {
		return nil, fmt.Errorf("Unable to create Google Cloud DNS service: %v", err)
	}
	return &DNSProviderGoogleCloud{
		project: project,
		client:  svc,
	}, nil
}

// Present creates a TXT record to fulfil the dns-01 challenge.
func (c *DNSProviderGoogleCloud) Present(domain, token, keyAuth string) error {
	fqdn, value, ttl := DNS01Record(domain, keyAuth)

	zone, err := c.getHostedZone(domain)
	if err != nil {
		return err
	}

	rec := &dns.ResourceRecordSet{
		Name:    fqdn,
		Rrdatas: []string{value},
		Ttl:     int64(ttl),
		Type:    "TXT",
	}
	change := &dns.Change{
		Additions: []*dns.ResourceRecordSet{rec},
	}

	chg, err := c.client.Changes.Create(c.project, zone, change).Do()
	if err != nil {
		return err
	}

	// wait for change to be completed
	for chg.Status == "pending" {
		time.Sleep(time.Second)

		chg, err = c.client.Changes.Get(c.project, zone, chg.Id).Do()
		if err != nil {
			return err
		}
	}

	// check DNS propagation here, because the default propagation timeout is too short.
	err = waitFor(120, 2, func() (bool, error) {
		return preCheckDNS(fqdn, value)
	})
	if err != nil {
		return err
	}

	return nil
}

// CleanUp removes the TXT record matching the specified parameters.
func (c *DNSProviderGoogleCloud) CleanUp(domain, token, keyAuth string) error {
	fqdn, _, _ := DNS01Record(domain, keyAuth)

	zone, err := c.getHostedZone(domain)
	if err != nil {
		return err
	}

	records, err := c.findTxtRecords(zone, fqdn)
	if err != nil {
		return err
	}

	for _, rec := range records {
		change := &dns.Change{
			Deletions: []*dns.ResourceRecordSet{rec},
		}
		_, err = c.client.Changes.Create(c.project, zone, change).Do()
		if err != nil {
			return err
		}
	}
	return nil
}

// getHostedZone returns the managed-zone
func (c *DNSProviderGoogleCloud) getHostedZone(domain string) (string, error) {

	zones, err := c.client.ManagedZones.List(c.project).Do()
	if err != nil {
		return "", fmt.Errorf("GoogleCloud API call failed: %v", err)
	}

	for _, z := range zones.ManagedZones {
		if strings.HasSuffix(domain+".", z.DnsName) {
			return z.Name, nil
		}
	}

	return "", fmt.Errorf("No matching GoogleCloud domain found for domain %s", domain)

}

func (c *DNSProviderGoogleCloud) findTxtRecords(zone, fqdn string) ([]*dns.ResourceRecordSet, error) {

	recs, err := c.client.ResourceRecordSets.List(c.project, zone).Do()
	if err != nil {
		return nil, err
	}

	found := []*dns.ResourceRecordSet{}
	for _, r := range recs.Rrsets {
		if r.Type == "TXT" && r.Name == fqdn {
			found = append(found, r)
		}
	}

	return found, nil
}

func gcloudEnvAuth() (gcloud string) {
	return os.Getenv("GCE_PROJECT")
}
