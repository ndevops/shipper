package capacity

import (
	"github.com/bookingcom/shipper/pkg/controller/capacity/builder"
	core_v1 "k8s.io/api/core/v1"
	"sort"

	shipper_v1alpha1 "github.com/bookingcom/shipper/pkg/apis/shipper/v1alpha1"
)

func buildReport(ownerName string, podsList []*core_v1.Pod) *shipper_v1alpha1.ClusterCapacityReport {

	sort.Slice(podsList, func(i, j int) bool {
		return podsList[i].Name < podsList[j].Name
	})

	reportBuilder := builder.NewReport(ownerName)

	for _, pod := range podsList {
		reportBuilder.AddPod(pod)
	}

	return reportBuilder.Build()
}
