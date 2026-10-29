package observability

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.37.0"
)

// serviceAccountDir is the directory of the service account files in a pod.
const serviceAccountDir = "/var/run/secrets/kubernetes.io/serviceaccount"

// K8sAttributesDetector detects the metadata of the Traefik pod running in a Kubernetes cluster.
// It reads the pod name from the hostname file and the namespace from the service account namespace file and queries the Kubernetes API to get the pod's UID.
// It queries the API with the standard library: the Kubernetes client would be the largest dependency of Traefik for one request.
type K8sAttributesDetector struct{}

func (K8sAttributesDetector) Detect(ctx context.Context) (*resource.Resource, error) {
	attrs := os.Getenv("OTEL_RESOURCE_ATTRIBUTES")
	if strings.Contains(attrs, string(semconv.K8SPodNameKey)) || strings.Contains(attrs, string(semconv.K8SPodUIDKey)) {
		return resource.Empty(), nil
	}

	// The same detection as the in-cluster configuration of the Kubernetes client.
	apiHost, apiPort := os.Getenv("KUBERNETES_SERVICE_HOST"), os.Getenv("KUBERNETES_SERVICE_PORT")
	if apiHost == "" || apiPort == "" {
		return resource.Empty(), nil
	}

	podName, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("getting pod name: %w", err)
	}

	podNamespaceBytes, err := os.ReadFile(path.Join(serviceAccountDir, "namespace"))
	if err != nil {
		return nil, fmt.Errorf("getting pod namespace: %w", err)
	}
	podNamespace := string(podNamespaceBytes)

	client, token, err := newInClusterClient()
	if err != nil {
		return nil, err
	}

	podUID, err := getPodUID(ctx, client, "https://"+net.JoinHostPort(apiHost, apiPort), token, podNamespace, podName)
	if errors.Is(err, errPodForbiddenOrNotFound) {
		log.Error().Err(err).Msg("Unable to build K8s resource attributes for Traefik pod")
		return resource.Empty(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("getting pod metadata: %w", err)
	}

	// To avoid version conflicts with other detectors, we use a Schemaless resource.
	return resource.NewSchemaless(
		semconv.K8SPodUID(podUID),
		semconv.K8SPodName(podName),
		semconv.K8SNamespaceName(podNamespace),
	), nil
}

var errPodForbiddenOrNotFound = errors.New("pod forbidden or not found")

// newInClusterClient returns the HTTP client and the token of the service account of the pod.
func newInClusterClient() (*http.Client, string, error) {
	token, err := os.ReadFile(path.Join(serviceAccountDir, "token"))
	if err != nil {
		return nil, "", fmt.Errorf("reading service account token: %w", err)
	}

	caCert, err := os.ReadFile(path.Join(serviceAccountDir, "ca.crt"))
	if err != nil {
		return nil, "", fmt.Errorf("reading service account CA: %w", err)
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCert) {
		return nil, "", errors.New("invalid service account CA")
	}

	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: pool}},
	}

	return client, strings.TrimSpace(string(token)), nil
}

// getPodUID gets the UID of the pod from the Kubernetes API.
func getPodUID(ctx context.Context, client *http.Client, apiURL, token, namespace, name string) (string, error) {
	podURL := apiURL + "/api/v1/namespaces/" + url.PathEscape(namespace) + "/pods/" + url.PathEscape(name)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, podURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusNotFound:
		return "", fmt.Errorf("%w: %s", errPodForbiddenOrNotFound, resp.Status)
	default:
		return "", fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var pod struct {
		Metadata struct {
			UID string `json:"uid"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pod); err != nil {
		return "", fmt.Errorf("decoding pod: %w", err)
	}

	return pod.Metadata.UID, nil
}
