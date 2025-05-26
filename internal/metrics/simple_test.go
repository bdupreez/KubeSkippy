package metrics

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	metricsfake "k8s.io/metrics/pkg/client/clientset/versioned/fake"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
)

func TestNewCollector(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	ctrlClient := ctrlclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()
	metricsClient := metricsfake.NewSimpleClientset()

	collector := NewCollector(ctrlClient, clientset, metricsClient)
	assert.NotNil(t, collector)
	assert.NotNil(t, collector.client)
	assert.NotNil(t, collector.clientset)
	assert.NotNil(t, collector.metricsClient)
}

func TestCollectMetrics_Basic(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = v1alpha1.AddToScheme(scheme)

	ctrlClient := ctrlclient.NewClientBuilder().WithScheme(scheme).Build()
	clientset := fake.NewSimpleClientset()
	metricsClient := metricsfake.NewSimpleClientset()

	collector := NewCollector(ctrlClient, clientset, metricsClient)

	policy := &v1alpha1.HealingPolicy{
		Spec: v1alpha1.HealingPolicySpec{
			Selector: v1alpha1.ResourceSelector{
				Namespaces: []string{"default"},
			},
		},
	}

	metrics, err := collector.CollectMetrics(context.Background(), policy)
	assert.NoError(t, err)
	assert.NotNil(t, metrics)
	assert.NotNil(t, metrics.Timestamp)
	assert.NotNil(t, metrics.Resources)
	assert.NotNil(t, metrics.Custom)
}