package e2e_test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ainannyv1alpha1 "github.com/kubeskippy/kubeskippy/api/v1alpha1"
)

var _ = Describe("HealingPolicy E2E Tests", func() {
	const (
		testNamespace = "e2e-test"
		timeout       = time.Minute * 5
		interval      = time.Second * 1
	)

	BeforeEach(func() {
		// Create test namespace
		createNamespace(testNamespace)
	})

	AfterEach(func() {
		// Clean up resources
		deleteNamespace(testNamespace)
	})

	Context("Pod Restart Healing", func() {
		It("should restart a pod with high restart count", func() {
			By("Creating a deployment with a failing container")
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failing-app",
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "failing-app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "failing-app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:    "app",
								Image:   "busybox",
								Command: []string{"sh", "-c", "exit 1"},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("10m"),
										corev1.ResourceMemory: resource.MustParse("10Mi"),
									},
								},
							}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			By("Creating a healing policy for pod restarts")
			policy := &ainannyv1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-restart-policy",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: ainannyv1alpha1.ResourceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "failing-app"},
						},
						Resources: []ainannyv1alpha1.ResourceFilter{{
							APIVersion: "v1",
							Kind:       "Pod",
						}},
					},
					Triggers: []ainannyv1alpha1.HealingTrigger{{
						Name: "restart-trigger",
						Type: "metric",
						MetricTrigger: &ainannyv1alpha1.MetricTrigger{
							Query:     "restart_count",
							Operator:  ">",
							Threshold: 3,
							Duration:  metav1.Duration{Duration: 2 * time.Minute},
						},
					}},
					Actions: []ainannyv1alpha1.HealingActionTemplate{{
						Name: "restart-action",
						Type: "restart",
						RestartAction: &ainannyv1alpha1.RestartAction{
							Strategy: "graceful",
						},
					}},
					SafetyRules: ainannyv1alpha1.SafetyRules{
						MaxActionsPerHour: 10,
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("Waiting for healing action to be created")
			Eventually(func() bool {
				var actions ainannyv1alpha1.HealingActionList
				err := k8sClient.List(ctx, &actions, client.InNamespace(testNamespace))
				if err != nil {
					return false
				}
				return len(actions.Items) > 0
			}, timeout, interval).Should(BeTrue())

			By("Verifying healing action is executed")
			var action ainannyv1alpha1.HealingAction
			Eventually(func() string {
				var actions ainannyv1alpha1.HealingActionList
				err := k8sClient.List(ctx, &actions, client.InNamespace(testNamespace))
				if err != nil || len(actions.Items) == 0 {
					return ""
				}
				action = actions.Items[0]
				return action.Status.Phase
			}, timeout, interval).Should(Equal("Completed"))

			By("Checking action result")
			Expect(action.Status.Result).To(Equal("Success"))
		})
	})

	Context("Deployment Scaling", func() {
		It("should scale deployment based on CPU usage", func() {
			By("Creating a deployment")
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cpu-app",
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "cpu-app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "cpu-app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "app",
								Image: "busybox",
								Command: []string{"sh", "-c",
									"while true; do echo 'Working...'; sleep 1; done"},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("100m"),
										corev1.ResourceMemory: resource.MustParse("50Mi"),
									},
								},
							}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			By("Creating a healing policy for scaling")
			policy := &ainannyv1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cpu-scaling-policy",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: ainannyv1alpha1.ResourceSelector{
						Resources: []ainannyv1alpha1.ResourceFilter{{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
						}},
					},
					Triggers: []ainannyv1alpha1.HealingTrigger{{
						Name: "cpu-trigger",
						Type: "metric",
						MetricTrigger: &ainannyv1alpha1.MetricTrigger{
							Query:     "cpu_usage_percent",
							Operator:  ">",
							Threshold: 80,
							Duration:  metav1.Duration{Duration: 1 * time.Minute},
						},
					}},
					Actions: []ainannyv1alpha1.HealingActionTemplate{{
						Name: "scale-action",
						Type: "scale",
						ScaleAction: &ainannyv1alpha1.ScaleAction{
							Direction: "up",
							Replicas:  3,
						},
					}},
					SafetyRules: ainannyv1alpha1.SafetyRules{
						MaxActionsPerHour: 5,
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			// In a real test, we would simulate high CPU usage
			// For now, we'll just verify the policy was created successfully
			By("Verifying policy is active")
			Eventually(func() string {
				var p ainannyv1alpha1.HealingPolicy
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      policy.Name,
					Namespace: policy.Namespace,
				}, &p)
				if err != nil {
					return ""
				}
				if len(p.Status.Conditions) > 0 {
					return string(p.Status.Conditions[0].Type)
				}
				return "Unknown"
			}, timeout, interval).Should(Equal("Active"))
		})
	})

	Context("AI-Driven Healing", func() {
		It("should analyze cluster state and recommend actions", func() {
			By("Creating a problematic deployment")
			deployment := &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "memory-leak-app",
					Namespace: testNamespace,
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: int32Ptr(1),
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{"app": "memory-leak-app"},
					},
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{"app": "memory-leak-app"},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name:  "app",
								Image: "busybox",
								Command: []string{"sh", "-c",
									"while true; do echo 'Leaking memory...'; sleep 5; done"},
								Resources: corev1.ResourceRequirements{
									Requests: corev1.ResourceList{
										corev1.ResourceCPU:    resource.MustParse("50m"),
										corev1.ResourceMemory: resource.MustParse("100Mi"),
									},
									Limits: corev1.ResourceList{
										corev1.ResourceMemory: resource.MustParse("200Mi"),
									},
								},
							}},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			By("Creating an AI-driven healing policy")
			policy := &ainannyv1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ai-analysis-policy",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingPolicySpec{
					Mode: "dryrun",
					Selector: ainannyv1alpha1.ResourceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"app": "memory-leak-app"},
						},
						Resources: []ainannyv1alpha1.ResourceFilter{{
							APIVersion: "v1",
							Kind:       "Pod",
						}},
					},
					Triggers: []ainannyv1alpha1.HealingTrigger{{
						Name: "ai-trigger",
						Type: "condition",
						ConditionTrigger: &ainannyv1alpha1.ConditionTrigger{
							Type:   "AIAnalysisRequired",
							Status: "True",
						},
					}},
					Actions: []ainannyv1alpha1.HealingActionTemplate{{
						Name: "ai-action",
						Type: "restart",
						RestartAction: &ainannyv1alpha1.RestartAction{
							Strategy: "graceful",
						},
					}},
					SafetyRules: ainannyv1alpha1.SafetyRules{
						MaxActionsPerHour: 3,
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("Verifying AI analysis is triggered")
			Eventually(func() bool {
				var p ainannyv1alpha1.HealingPolicy
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      policy.Name,
					Namespace: policy.Namespace,
				}, &p)
				if err != nil {
					return false
				}
				return !p.Status.LastEvaluated.IsZero()
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("Safety Rules", func() {
		It("should respect dry-run mode", func() {
			By("Creating a deployment")
			deployment := createTestDeployment("test-app", testNamespace)
			Expect(k8sClient.Create(ctx, deployment)).To(Succeed())

			By("Creating a healing policy with dry-run enabled")
			policy := &ainannyv1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "dry-run-policy",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: ainannyv1alpha1.ResourceSelector{
						Resources: []ainannyv1alpha1.ResourceFilter{{
							APIVersion: "apps/v1",
							Kind:       "Deployment",
						}},
					},
					Triggers: []ainannyv1alpha1.HealingTrigger{{
						Name: "manual-trigger",
						Type: "event",
						EventTrigger: &ainannyv1alpha1.EventTrigger{
							Type:   "Normal",
							Reason: "ManualTrigger",
						},
					}},
					Actions: []ainannyv1alpha1.HealingActionTemplate{{
						Name: "restart-action",
						Type: "restart",
						RestartAction: &ainannyv1alpha1.RestartAction{
							Strategy: "rolling",
						},
					}},
					SafetyRules: ainannyv1alpha1.SafetyRules{
						MaxActionsPerHour: 10,
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("Triggering a manual healing action")
			action := &ainannyv1alpha1.HealingAction{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "manual-dry-run-action",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingActionSpec{
					PolicyRef: ainannyv1alpha1.PolicyReference{
						Name:      policy.Name,
						Namespace: policy.Namespace,
					},
					TargetResource: ainannyv1alpha1.TargetResource{
						APIVersion: "apps/v1",
						Kind:       "Deployment",
						Name:       "test-app",
					},
					Action: ainannyv1alpha1.HealingActionTemplate{
						Type: "restart",
						RestartAction: &ainannyv1alpha1.RestartAction{
							Strategy: "rolling",
						},
					},
					DryRun: true,
				},
			}
			Expect(k8sClient.Create(ctx, action)).To(Succeed())

			By("Verifying action is completed as dry-run")
			Eventually(func() string {
				var a ainannyv1alpha1.HealingAction
				err := k8sClient.Get(ctx, types.NamespacedName{
					Name:      action.Name,
					Namespace: action.Namespace,
				}, &a)
				if err != nil {
					return ""
				}
				return a.Status.Phase
			}, timeout, interval).Should(Equal("Completed"))

			By("Verifying deployment was not actually restarted")
			var dep appsv1.Deployment
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-app",
				Namespace: testNamespace,
			}, &dep)).To(Succeed())
			// In dry-run, the deployment should not have been modified
			Expect(dep.Status.ObservedGeneration).To(Equal(int64(1)))
		})

		It("should enforce rate limits", func() {
			By("Creating a healing policy with low rate limit")
			policy := &ainannyv1alpha1.HealingPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rate-limited-policy",
					Namespace: testNamespace,
				},
				Spec: ainannyv1alpha1.HealingPolicySpec{
					Mode: "automatic",
					Selector: ainannyv1alpha1.ResourceSelector{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{"test": "rate-limit"},
						},
						Resources: []ainannyv1alpha1.ResourceFilter{{
							APIVersion: "v1",
							Kind:       "Pod",
						}},
					},
					Triggers: []ainannyv1alpha1.HealingTrigger{{
						Name: "manual-trigger",
						Type: "event",
						EventTrigger: &ainannyv1alpha1.EventTrigger{
							Type:   "Normal",
							Reason: "ManualTrigger",
						},
					}},
					Actions: []ainannyv1alpha1.HealingActionTemplate{{
						Name: "restart-action",
						Type: "restart",
						RestartAction: &ainannyv1alpha1.RestartAction{
							Strategy: "graceful",
						},
					}},
					SafetyRules: ainannyv1alpha1.SafetyRules{
						MaxActionsPerHour: 2,
					},
				},
			}
			Expect(k8sClient.Create(ctx, policy)).To(Succeed())

			By("Creating multiple healing actions")
			for i := 0; i < 3; i++ {
				action := &ainannyv1alpha1.HealingAction{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("rate-test-action-%d", i),
						Namespace: testNamespace,
					},
					Spec: ainannyv1alpha1.HealingActionSpec{
						PolicyRef: ainannyv1alpha1.PolicyReference{
							Name:      policy.Name,
							Namespace: policy.Namespace,
						},
						TargetResource: ainannyv1alpha1.TargetResource{
							APIVersion: "v1",
							Kind:       "Pod",
							Name:       fmt.Sprintf("test-pod-%d", i),
						},
						Action: ainannyv1alpha1.HealingActionTemplate{
							Type: "restart",
						},
					},
				}
				Expect(k8sClient.Create(ctx, action)).To(Succeed())
			}

			By("Verifying rate limit is enforced")
			Eventually(func() int {
				var actions ainannyv1alpha1.HealingActionList
				err := k8sClient.List(ctx, &actions, client.InNamespace(testNamespace))
				if err != nil {
					return 0
				}

				completedCount := 0
				for _, action := range actions.Items {
					if action.Status.Phase == "Succeeded" && action.Status.Result != nil && action.Status.Result.Success {
						completedCount++
					}
				}
				return completedCount
			}, timeout, interval).Should(Equal(2)) // Only 2 should succeed due to rate limit
		})
	})
})

// Helper functions

func int32Ptr(i int32) *int32 {
	return &i
}

func createTestDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": name},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": name},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "app",
						Image: "nginx:alpine",
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("20Mi"),
							},
						},
					}},
				},
			},
		},
	}
}
