package installation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"

	shipper "github.com/bookingcom/shipper/pkg/apis/shipper/v1alpha1"
	shippertesting "github.com/bookingcom/shipper/pkg/testing"
)

var (
	TargetConditionOperational = shipper.TargetCondition{
		Type:   shipper.TargetConditionTypeOperational,
		Status: corev1.ConditionTrue,
	}
	TargetConditionReady = shipper.TargetCondition{
		Type:   shipper.TargetConditionTypeReady,
		Status: corev1.ConditionTrue,
	}
	TargetConditionReadyUnknown = shipper.TargetCondition{
		Type:   shipper.TargetConditionTypeReady,
		Status: corev1.ConditionUnknown,
	}

	SuccessStatus = shipper.InstallationTargetStatus{
		Conditions: []shipper.TargetCondition{
			TargetConditionOperational,
			TargetConditionReady,
		},
	}

	apiResourceList = []*metav1.APIResourceList{
		{
			GroupVersion: "v1",
			APIResources: []metav1.APIResource{
				{
					Kind:       "Service",
					Namespaced: true,
					Name:       "services",
				},
			},
		},
		{
			GroupVersion: "apps/v1",
			APIResources: []metav1.APIResource{
				{
					Kind:       "Deployment",
					Namespaced: true,
					Name:       "deployments",
				},
			},
		},
	}
)

var localFetchChart = func(chartspec *shipper.Chart) (*chart.Chart, error) {
	re := regexp.MustCompile(`[^a-zA-Z0-9]+`)
	pathurl := re.ReplaceAllString(chartspec.RepoURL, "_")
	data, err := ioutil.ReadFile(
		path.Join(
			"testdata",
			"chart-cache",
			pathurl,
			fmt.Sprintf("%s-%s.tgz", chartspec.Name, chartspec.Version),
		))
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(data)
	return chartutil.LoadArchive(buf)
}

func newFixture(objects []runtime.Object) *shippertesting.ControllerTestFixture {
	f := shippertesting.NewControllerTestFixture()
	f.InitializeDiscovery(apiResourceList)
	f.InitializeDynamicClient(objects)

	return f
}

func buildInstallationTarget(namespace, appName string, chart shipper.Chart) *shipper.InstallationTarget {
	return &shipper.InstallationTarget{
		ObjectMeta: v1.ObjectMeta{
			Name:      appName,
			Namespace: namespace,
			Labels: map[string]string{
				shipper.AppLabel:            appName,
				shipper.HelmWorkaroundLabel: shipper.True,
			},
		},
		Spec: shipper.InstallationTargetSpec{
			Chart:       chart,
			Values:      nil,
			CanOverride: true,
		},
	}
}

func buildChart(appName, version, repoUrl string) shipper.Chart {
	return shipper.Chart{
		Name:    appName,
		Version: version,
		RepoURL: repoUrl,
	}
}
