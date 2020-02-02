package kotsadm

import (
	"fmt"
	"github.com/replicatedhq/kots/pkg/kotsadm/hostnetwork"
	"time"

	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func migrationsPod(deployOptions types.DeployOptions) *corev1.Pod {
	name := fmt.Sprintf("kotsadm-migrations-%d", time.Now().Unix())

	var securityContext corev1.PodSecurityContext
	if !deployOptions.IsOpenShift {
		securityContext = corev1.PodSecurityContext{
			RunAsUser: util.IntPointer(1001),
			FSGroup:   util.IntPointer(1001),
		}
	}

	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: deployOptions.Namespace,
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
			},
		},
		Spec: corev1.PodSpec{
			Tolerations:     hostnetwork.Tolerations(deployOptions.UseHostNetwork),
			HostNetwork:     deployOptions.UseHostNetwork,
			SecurityContext: &securityContext,
			RestartPolicy:   corev1.RestartPolicyOnFailure,
			Containers: []corev1.Container{
				{
					Image:           fmt.Sprintf("%s/kotsadm-migrations:%s", kotsadmRegistry(), kotsadmTag()),
					ImagePullPolicy: corev1.PullAlways,
					Name:            name,
					Env: []corev1.EnvVar{
						{
							Name:  "SCHEMAHERO_DRIVER",
							Value: "postgres",
						},
						{
							Name:  "SCHEMAHERO_SPEC_FILE",
							Value: "/tables",
						},
						{
							Name: "SCHEMAHERO_URI",
							ValueFrom: &corev1.EnvVarSource{
								SecretKeyRef: &corev1.SecretKeySelector{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: "kotsadm-postgres",
									},
									Key: "uri",
								},
							},
						},
					},
				},
			},
		},
	}

	return pod
}
