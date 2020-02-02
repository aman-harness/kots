package kotsadm

import (
	"bytes"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getPostgresYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var statefulset bytes.Buffer
	if deployOptions.PostgresPassword == "" {
		deployOptions.PostgresPassword = uuid.New().String()
	}
	if err := s.Encode(postgresStatefulset(deployOptions), &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres statefulset")
	}
	docs["postgres-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(postgresService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres service")
	}
	docs["postgres-service.yaml"] = service.Bytes()

	var hostpathVolume bytes.Buffer
	if deployOptions.UseHostNetwork {
		if err := s.Encode(postgresHostpathVolume(), &hostpathVolume); err != nil {
			return nil, errors.Wrap(err, "failed to marshal postgres hostPath persistent volume")
		}
		docs["postgres-pv.yaml"] = hostpathVolume.Bytes()
	}

	return docs, nil
}

func ensurePostgres(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensurePostgresSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	if err := ensurePostgresStatefulset(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres statefulset")
	}

	if err := ensurePostgresService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres service")
	}

	return nil
}

func ensurePostgresStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(postgresStatefulset(deployOptions))
		if err != nil {
			return errors.Wrap(err, "failed to create postgres statefulset")
		}

		if deployOptions.UseHostNetwork {
			_, err := clientset.CoreV1().PersistentVolumes().Create(postgresHostpathVolume())
			if err != nil {
				return errors.Wrap(err, "failed to create postgres hostpath persistentvolume")
			}
		}
	}

	return nil
}

func ensurePostgresService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get("kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(postgresService(namespace))
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}
