package main

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	cdn "git.arvancloud.ir/arvancloud/cdn-go-sdk"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	metav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kubemetav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

const (
	defaultTTL = 120 // Default TTL for DNS records in seconds
)

var (
	GroupName            = cmp.Or(os.Getenv("GROUP_NAME"), "acme.arvancloud.ir")
	ArvancloudCDNBaseURL = cmp.Or(os.Getenv("ARVANCLOUD_CDN_BASE_URL"), "https://napi.arvancloud.ir/cdn/4.0")
)

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	// This will register our Arvancloud DNS provider with the webhook server
	cmd.RunWebhookServer(GroupName,
		&arvancloudProviderSolver{},
	)
}

// arvancloudProviderSolver implements the provider-specific logic needed to
// solve DNS01 challenges using Arvancloud.
type arvancloudProviderSolver struct {
	kubeClient *kubernetes.Clientset
}

// Name returns the name of the DNS provider
func (c *arvancloudProviderSolver) Name() string {
	return "arvancloud"
}

// Initialize will be called when the webhook first starts.
func (c *arvancloudProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}

	c.kubeClient = cl
	return nil
}

// Present is responsible for creating the DNS record for domain validation
func (c *arvancloudProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	klog.InfoS("Presenting DNS01 challenge", "domain", ch.DNSName)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	// Get Arvancloud API key from the provided secret
	apiKey, err := cfg.getAPIKey(ch.ResourceNamespace, c.kubeClient)
	if err != nil {
		return fmt.Errorf("unable to get API key: %v", err)
	}

	// Create a new Arvancloud CDN API client
	apiClient, err := NewArvancloudCDNAPIClient(apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Arvancloud CDN API client: %v", err)
	}

	// Remove trailing dot from domain
	domain := util.UnFqdn(ch.ResolvedZone)

	// Create the TXT record
	recordName := extractRecordName(ch.ResolvedFQDN, domain)

	record := cdn.NewTXTRecord()
	record.SetName(recordName)
	record.SetTtl(defaultTTL)
	record.SetType("TXT")
	record.Value = cdn.NewTXTRecordValue(ch.Key)

	res, _, err := apiClient.DNSManagementAPI.DnsRecordsStore(context.TODO(), domain).DnsRecord(cdn.DnsRecord{
		TXTRecord: record,
	}).Execute()

	if err != nil {
		return fmt.Errorf("failed to create DNS record: %v", err)
	}

	klog.InfoS("DNS record created successfully", "domain", domain, "recordName", recordName, "message", res.Message)

	return nil
}

// CleanUp is responsible for cleaning up the DNS record after validation
func (c *arvancloudProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	klog.InfoS("Cleaning up DNS01 challenge", "domain", ch.DNSName)

	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return fmt.Errorf("unable to load config: %v", err)
	}

	/// Get Arvancloud API key from the provided secret
	apiKey, err := cfg.getAPIKey(ch.ResourceNamespace, c.kubeClient)
	if err != nil {
		return fmt.Errorf("unable to get API key: %v", err)
	}

	// Create a new Arvancloud CDN API client
	apiClient, err := NewArvancloudCDNAPIClient(apiKey)
	if err != nil {
		return fmt.Errorf("failed to create Arvancloud CDN API client: %v", err)
	}

	// Remove trailing dot from domain
	domain := util.UnFqdn(ch.ResolvedZone)

	// Create the TXT record
	recordName := extractRecordName(ch.ResolvedFQDN, domain)

	indexRes, _, err := apiClient.DNSManagementAPI.DnsRecordsIndex(context.TODO(), domain).Type_("TXT").Search(recordName).Execute()
	if err != nil {
		return fmt.Errorf("failed to find DNS record: %v", err)
	}

	for _, record := range indexRes.Data {
		r := record.DnsRecordGenericObjectValue
		if r.GetName() != recordName {
			continue
		}

		if r.GetValue()["text"].(string) != ch.Key {
			continue
		}

		res, _, err := apiClient.DNSManagementAPI.DnsRecordsDestroy(context.TODO(), domain, r.GetId()).Execute()
		if err != nil {
			return fmt.Errorf("failed to delete DNS record: %v", err)
		}

		klog.InfoS("DNS record deleted successfully", "domain", domain, "recordName", recordName, "message", res.Message)
		return nil
	}

	klog.Warningf("DNS record not found for cleanup domain: %s, recordName: %s", domain, recordName)

	return nil
}

// extractRecordName extracts the record name from the FQDN by removing the domain part
func extractRecordName(fqdn, domain string) string {
	name := util.UnFqdn(fqdn)
	if name == domain {
		return "@"
	}
	return name[:len(name)-len(domain)-1]
}

// loadConfig loads the Arvancloud provider configuration
type arvancloudProviderConfig struct {
	APIKey          string                   `json:"apiKey"`
	APIKeySecretRef metav1.SecretKeySelector `json:"apiKeySecretRef"`
	TTL             int                      `json:"ttl"`
}

func (cfg *arvancloudProviderConfig) getAPIKey(namespace string, client *kubernetes.Clientset) (string, error) {
	if cfg.APIKey != "" {
		return cfg.APIKey, nil
	}
	if cfg.APIKeySecretRef.LocalObjectReference.Name == "" {
		return "", fmt.Errorf("one of apiKey or apiKeySecretRef should be provided")
	}
	secret, err := client.CoreV1().Secrets(namespace).Get(context.TODO(), cfg.APIKeySecretRef.LocalObjectReference.Name, kubemetav1.GetOptions{})
	if err != nil {
		return "", err
	}
	data, ok := secret.Data[cfg.APIKeySecretRef.Key]
	if !ok {
		return "", fmt.Errorf("key %v not found is %v/%v", cfg.APIKeySecretRef.Key, namespace, cfg.APIKeySecretRef.LocalObjectReference.Name)
	}
	return string(data), nil
}

func loadConfig(cfgJSON *extapi.JSON) (*arvancloudProviderConfig, error) {
	if cfgJSON == nil {
		return nil, fmt.Errorf("configuration not provided")
	}

	cfg := &arvancloudProviderConfig{}
	if err := json.Unmarshal(cfgJSON.Raw, cfg); err != nil {
		return nil, fmt.Errorf("error decoding solver config: %v", err)
	}

	// Set default TTL if not provided
	if cfg.TTL <= 0 {
		cfg.TTL = defaultTTL
	}

	return cfg, nil
}

func NewArvancloudCDNAPIClient(apiKey string) (*cdn.APIClient, error) {
	configuration := cdn.NewConfiguration()
	configuration.AddDefaultHeader("Authorization", "apikey "+strings.TrimLeft(strings.ToLower(apiKey), "apikey "))
	configuration.UserAgent = "mohammadv184/cert-manager-arvancloud" + " (" + fmt.Sprintf("Version: %s;BuildDate: %s;Commit: %s;OS/Arch: %s/%s;",
		Version, BuildDate, Commit, OS, Arch) + ")"
	configuration.Servers[0].URL = ArvancloudCDNBaseURL

	client := cdn.NewAPIClient(configuration)
	if client == nil {
		return nil, fmt.Errorf("failed to create Arvancloud CDN API client")
	}

	return client, nil
}
