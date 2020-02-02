package kotsadm

import (
	"fmt"
	"github.com/replicatedhq/kots/pkg/kotsadm/hostnetwork"

	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func kotsadmRole(namespace string) *rbacv1.Role {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-role",
			Namespace: namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
		// creation cannot be restricted by name
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups:     []string{""},
				Resources:     []string{"configmaps"},
				ResourceNames: []string{"kotsadm-application-metadata", "kotsadm-gitops"},
				Verbs:         metav1.Verbs{"get", "delete", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     metav1.Verbs{"create"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				ResourceNames: []string{
					"kotsadm-encryption",
					"kotsadm-gitops",
					"kotsadm-password",
					auth.KotsadmAuthstringSecretName,
				},
				Verbs: metav1.Verbs{"get", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     metav1.Verbs{"create"},
			},
		},
	}

	return role
}

func kotsadmRoleBinding(namespace string) *rbacv1.RoleBinding {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rolebinding",
			Namespace: namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "kotsadm",
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     "kotsadm-role",
		},
	}

	return roleBinding
}

func kotsadmServiceAccount(namespace string) *corev1.ServiceAccount {
	serviceAccount := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
	}

	return serviceAccount
}

func kotsadmDeployment(deployOptions types.DeployOptions) *appsv1.Deployment {
	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
		}
	}

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: deployOptions.Namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "kotsadm",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":            "kotsadm",
						types.KotsadmKey: types.KotsadmLabelValue,
					},
				},
				Spec: corev1.PodSpec{
					Tolerations:        hostnetwork.Tolerations(deployOptions.UseHostNetwork),
					HostNetwork:        deployOptions.UseHostNetwork,
					SecurityContext:    &securityContext,
					ServiceAccountName: "kotsadm",
					RestartPolicy:      corev1.RestartPolicyAlways,
					Containers: []corev1.Container{
						{
							Image:           fmt.Sprintf("%s/kotsadm:%s", kotsadmRegistry(), kotsadmTag()),
							ImagePullPolicy: corev1.PullAlways,
							Name:            "kotsadm",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: hostnetwork.ContainerPorts(deployOptions.UseHostNetwork).KotsadmKotsadm,
									HostPort:      hostnetwork.HostPorts(deployOptions.UseHostNetwork).KotsadmKotsadm,
								},
							},
							ReadinessProbe: &corev1.Probe{
								FailureThreshold:    3,
								InitialDelaySeconds: 10,
								PeriodSeconds:       10,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   "/healthz",
										Port:   intstr.FromInt(3000),
										Scheme: corev1.URISchemeHTTP,
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "SHARED_PASSWORD_BCRYPT",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-password",
											},
											Key: "passwordBcrypt",
										},
									},
								},
								{
									Name: "SESSION_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-session",
											},
											Key: "key",
										},
									},
								},
								{
									Name: "POSTGRES_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "kotsadm-postgres",
											},
											Key: "uri",
										},
									},
								},
								{
									Name: "POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "metadata.namespace",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return deployment
}

func kotsadmService(namespace string) *corev1.Service {
	port := corev1.ServicePort{
		Name:       "http",
		Port:       3000,
		TargetPort: intstr.FromString("http"),
	}

	serviceType := corev1.ServiceTypeClusterIP

	service := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm",
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"app": "kotsadm",
			},
			Type: serviceType,
			Ports: []corev1.ServicePort{
				port,
			},
		},
	}

	return service
}
