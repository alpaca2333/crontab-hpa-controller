package cron_hpa

import (
	"hash/fnv"
	"io/ioutil"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/api/autoscaling/v2beta2"
)

// GetCurrentNamesace Get current namespace of this release.
func GetCurrentNamesace() string {
	ns, _ := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	return string(ns)
}

// FindCorrespondingHpa Find the hpa object of which the target is the specified deployment.
func FindCorrespondingHpa(deploy v1.Deployment,
	hpas *v2beta2.HorizontalPodAutoscalerList) *v2beta2.HorizontalPodAutoscaler {
	if hpas == nil {
		return nil
	}
	for _, hpa := range hpas.Items {
		if hpa.Spec.ScaleTargetRef.Name == deploy.Name &&
			hpa.Spec.ScaleTargetRef.Kind == "Deployment" {
			return &hpa
		}
	}
	return nil
}

// Hash Generate the hash of some strings.
func Hash(args ...string) uint32 {
	h := fnv.New32a()
	for _, arg := range args {
		_, _ = h.Write([]byte(arg))
	}
	return h.Sum32()
}