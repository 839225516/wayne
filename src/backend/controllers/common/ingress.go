package common

import (
	kapiv1 "k8s.io/api/networking/v1"

	"wayne/src/backend/models"
)

func IngressPreDeploy(kubeIngress *kapiv1.Ingress, cluster *models.Cluster, namespace *models.Namespace) {
	preDefinedAnnotationMap := make(map[string]string)

	annotationResult := make(map[string]string, 0)
	// user defined
	for k, v := range kubeIngress.Annotations {
		preDefinedAnnotationMap[k] = v
	}
	// cluster defined, overwrite user defined
	for k, v := range cluster.MetaDataObj.IngressAnnotations {
		preDefinedAnnotationMap[k] = v
	}
	// namespace defined, overwrite cluster and user defined
	for k, v := range namespace.MetaDataObj.IngressAnnotations {
		preDefinedAnnotationMap[k] = v
	}
	for k, v := range preDefinedAnnotationMap {
		annotationResult[k] = v
	}

	kubeIngress.Annotations = annotationResult
}
