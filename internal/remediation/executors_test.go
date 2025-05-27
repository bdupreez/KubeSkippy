package remediation

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kubeskippy/kubeskippy/api/v1alpha1"
)

func TestRestartExecutor(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	tests := []struct {
		name           string
		target         client.Object
		action         *v1alpha1.HealingActionTemplate
		expectedResult bool
		expectedError  bool
	}{
		{
			name: "restart pod",
			target: createUnstructuredPod("test-pod", "default"),
			action: &v1alpha1.HealingActionTemplate{
				Type: "restart",
				RestartAction: &v1alpha1.RestartAction{
					Strategy: "rolling",
				},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name: "restart deployment",
			target: createUnstructuredDeployment("test-deployment", "default"),
			action: &v1alpha1.HealingActionTemplate{
				Type: "restart",
				RestartAction: &v1alpha1.RestartAction{
					Strategy: "rolling",
				},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name: "unsupported resource",
			target: createUnstructuredService("test-service", "default"),
			action: &v1alpha1.HealingActionTemplate{
				Type: "restart",
			},
			expectedResult: false,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(tt.target).
				Build()

			executor := NewRestartExecutor(fakeClient)

			// Test validation
			err := executor.Validate(context.Background(), tt.target, tt.action)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Test dry-run
			result, err := executor.DryRun(context.Background(), tt.target, tt.action)
			require.NoError(t, err)
			assert.True(t, result.Success)
			assert.Contains(t, result.Message, "Dry-run")

			// Test execution
			result, err = executor.Execute(context.Background(), tt.target, tt.action)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result.Success)
				assert.NotEmpty(t, result.Changes)
			}
		})
	}
}

func TestScaleExecutor(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = appsv1.AddToScheme(scheme)

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-deployment",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(3),
		},
	}

	tests := []struct {
		name           string
		action         *v1alpha1.HealingActionTemplate
		expectedResult bool
		expectedReplicas int32
	}{
		{
			name: "scale up",
			action: &v1alpha1.HealingActionTemplate{
				Type: "scale",
				ScaleAction: &v1alpha1.ScaleAction{
					Direction: "up",
					Replicas:  2,
				},
			},
			expectedResult: true,
			expectedReplicas: 5,
		},
		{
			name: "scale down",
			action: &v1alpha1.HealingActionTemplate{
				Type: "scale",
				ScaleAction: &v1alpha1.ScaleAction{
					Direction:   "down",
					Replicas:    1,
					MinReplicas: 1,
				},
			},
			expectedResult: true,
			expectedReplicas: 2,
		},
		{
			name: "scale absolute",
			action: &v1alpha1.HealingActionTemplate{
				Type: "scale",
				ScaleAction: &v1alpha1.ScaleAction{
					Direction: "absolute",
					Replicas:  10,
				},
			},
			expectedResult: true,
			expectedReplicas: 10,
		},
		{
			name: "scale with max limit",
			action: &v1alpha1.HealingActionTemplate{
				Type: "scale",
				ScaleAction: &v1alpha1.ScaleAction{
					Direction:   "up",
					Replicas:    10,
					MaxReplicas: 5,
				},
			},
			expectedResult: true,
			expectedReplicas: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset deployment replicas
			deploymentCopy := deployment.DeepCopy()
			
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(deploymentCopy).
				Build()

			executor := NewScaleExecutor(fakeClient)

			// Test validation
			err := executor.Validate(context.Background(), deploymentCopy, tt.action)
			require.NoError(t, err)

			// Test execution
			result, err := executor.Execute(context.Background(), deploymentCopy, tt.action)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.Success)
			
			// Verify replicas were updated
			var updated appsv1.Deployment
			err = fakeClient.Get(context.Background(), client.ObjectKey{
				Namespace: "default",
				Name:      "test-deployment",
			}, &updated)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedReplicas, *updated.Spec.Replicas)
		})
	}
}

func TestPatchExecutor(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-configmap",
			Namespace: "default",
		},
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
	}

	tests := []struct {
		name           string
		action         *v1alpha1.HealingActionTemplate
		expectedResult bool
		expectedData   map[string]string
	}{
		{
			name: "patch single field",
			action: &v1alpha1.HealingActionTemplate{
				Type: "patch",
				PatchAction: &v1alpha1.PatchAction{
					Type: "merge",
					Patches: []v1alpha1.PatchOperation{
						{
							Path:  []string{"data", "key1"},
							Value: "\"modified\"",
						},
					},
				},
			},
			expectedResult: true,
			expectedData: map[string]string{
				"key1": "modified",
				"key2": "value2",
			},
		},
		{
			name: "patch multiple fields",
			action: &v1alpha1.HealingActionTemplate{
				Type: "patch",
				PatchAction: &v1alpha1.PatchAction{
					Type: "merge",
					Patches: []v1alpha1.PatchOperation{
						{
							Path:  []string{"data", "key1"},
							Value: "\"changed1\"",
						},
						{
							Path:  []string{"data", "key3"},
							Value: "\"new\"",
						},
					},
				},
			},
			expectedResult: true,
			expectedData: map[string]string{
				"key1": "changed1",
				"key2": "value2",
				"key3": "new",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configMapCopy := configMap.DeepCopy()
			
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(configMapCopy).
				Build()

			executor := NewPatchExecutor(fakeClient)

			// Test validation
			err := executor.Validate(context.Background(), configMapCopy, tt.action)
			require.NoError(t, err)

			// Test execution
			result, err := executor.Execute(context.Background(), configMapCopy, tt.action)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result.Success)
			assert.NotEmpty(t, result.Changes)

			// Verify data was updated
			var updated corev1.ConfigMap
			err = fakeClient.Get(context.Background(), client.ObjectKey{
				Namespace: "default",
				Name:      "test-configmap",
			}, &updated)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedData, updated.Data)
		})
	}
}

func TestDeleteExecutor(t *testing.T) {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "test-namespace",
		},
	}

	criticalPod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "critical-pod",
			Namespace: "kube-system",
		},
	}

	tests := []struct {
		name           string
		target         client.Object
		action         *v1alpha1.HealingActionTemplate
		expectedResult bool
		expectedError  bool
	}{
		{
			name:   "delete pod",
			target: pod,
			action: &v1alpha1.HealingActionTemplate{
				Type: "delete",
				DeleteAction: &v1alpha1.DeleteAction{
					GracePeriodSeconds: 30,
				},
			},
			expectedResult: true,
			expectedError:  false,
		},
		{
			name:   "delete protected namespace",
			target: criticalPod,
			action: &v1alpha1.HealingActionTemplate{
				Type: "delete",
			},
			expectedResult: false,
			expectedError:  true,
		},
		{
			name:   "force delete with finalizers",
			target: pod,
			action: &v1alpha1.HealingActionTemplate{
				Type: "delete",
				DeleteAction: &v1alpha1.DeleteAction{
					Force: true,
				},
			},
			expectedResult: true,
			expectedError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targetCopy := tt.target.DeepCopyObject().(client.Object)
			
			fakeClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(targetCopy).
				Build()

			executor := NewDeleteExecutor(fakeClient)

			// Test validation
			err := executor.Validate(context.Background(), targetCopy, tt.action)
			if tt.expectedError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)

			// Test execution
			result, err := executor.Execute(context.Background(), targetCopy, tt.action)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result.Success)
			}
		})
	}
}

// Helper functions to create unstructured objects
func createUnstructuredPod(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Pod",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"containers": []interface{}{
					map[string]interface{}{
						"name":  "test",
						"image": "test:latest",
					},
				},
			},
		},
	}
}

func createUnstructuredDeployment(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"replicas": 3,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "test",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "test",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "test",
								"image": "test:latest",
							},
						},
					},
				},
			},
		},
	}
}

func createUnstructuredService(name, namespace string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": "test",
				},
				"ports": []interface{}{
					map[string]interface{}{
						"port":       80,
						"targetPort": 8080,
					},
				},
			},
		},
	}
}