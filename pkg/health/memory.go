package health

import (
	"context"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	mv1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"
)

// GetMemory returns the memory limit of the container in the pod specified by the name param
func GetMemory(name string) int64 {

	var podMetrics, err = mClient.MetricsV1alpha1().PodMetricses(namespace).Get(context.TODO(), name, mv1.GetOptions{})
	if err != nil {
		logrus.Error("Error getting metrics for pod: ", name)
		return -1
	}
	for _, container := range podMetrics.Containers {
		memory, ok := container.Usage.Memory().AsInt64()
		if !ok {
			logrus.Error("Error getting the memory usage of container")
		} else {
			return memory
		}
	}
	return -1
}

// AddMemory multiplies the existing memory limit of deployment by memFactor
func AddMemory(memFactor int, name string) {

	// If supplied memFactor is less than 1, we default to 2
	if memFactor < 1 {
		memFactor = 2
	}

	currMem := GetMemory(name)
	if currMem == -1 {
		logrus.Error("Memory limit value could not be fetched")
	}
	newMem := int(currMem) * memFactor

	// conflict might occur if the deployment gets updated while we're trying to modify it.
	// hence, retry on conflict is used.
	retryErr := retry.RetryOnConflict(retry.DefaultRetry, func() error {

		result, err := dClient.Get(context.TODO(), deployment, mv1.GetOptions{})
		if err != nil {
			logrus.Error("Error getting deployment :", err)
		}
		var i = 0
		var updateErr error

		// for all containers in the deployment, increase memory limit
		// TODO, use number of pods var instead of hardcoding
		for i < 2 {
			result.Spec.Template.Spec.Containers[i].Resources.Limits =
				make(map[v1.ResourceName]resource.Quantity)

			result.Spec.Template.Spec.Containers[i].Resources.Limits[v1.ResourceMemory] =
				resource.MustParse(strconv.Itoa(newMem))

			_, updateErr = dClient.Update(context.TODO(), result, mv1.UpdateOptions{})
			i = i + 1
		}
		return updateErr
	})

	if retryErr != nil {
		logrus.Error("Retry on conflict fails: ", retryErr)
	}

	// Sleep till all pods are running again
	for !PodsReady() {
		logrus.Info("Waiting for the pods to be up and running")
		time.Sleep(500 * time.Millisecond)
	}

}

// IsOutOfMemory checks the timestamp array of Pod restarts to figure out
// if the pods are running out of memory. If the restart times are too frequent we
// can assume that further restarts won't be helpful and so it is a memory issue.
func IsOutOfMemory(ts []time.Time) bool {

	return false
}
