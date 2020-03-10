package installation

import (
	"fmt"
	"regexp"
	"testing"

	shipper "github.com/bookingcom/shipper/pkg/apis/shipper/v1alpha1"
	shippererrors "github.com/bookingcom/shipper/pkg/errors"
	shippertesting "github.com/bookingcom/shipper/pkg/testing"
)

// TestRendererBrokenChartTarball tests if the renderer returns an error for a
// chart that points to a broken tarball.
func TestRendererBrokenChartTarball(t *testing.T) {
	it := buildInstallationTarget(
		shippertesting.TestNamespace,
		shippertesting.TestApp,
		buildChart("reviews-api", "broken-tarball", repoUrl))

	_, err := FetchAndRenderChart(localFetchChart, it)

	if err == nil {
		t.Fatal("FetchAndRenderChart should return error, invalid tarball")
	}

	if _, ok := err.(shippererrors.RenderManifestError); !ok {
		t.Fatalf("FetchAndRenderChart should fail with RenderManifestError, got %v instead", err)
	}

	t.Logf("FetchAndRenderChart failed as expected. errors was: %s", err.Error())
}

// TestRendererBrokenObjects tests if the renderer returns an error when the
// the chart has manifests that were encoded improperly.
func TestRendererBrokenObjects(t *testing.T) {
	it := buildInstallationTarget(
		shippertesting.TestNamespace,
		shippertesting.TestApp,
		buildChart("reviews-api", "broken-k8s-objects", repoUrl))

	_, err := FetchAndRenderChart(localFetchChart, it)

	if err == nil {
		t.Fatal("FetchAndRenderChart should return error, broken serialization")
	}

	if _, ok := err.(shippererrors.RenderManifestError); !ok {
		t.Fatalf("FetchAndRenderChart should fail with RenderManifestError, got %v instead", err)
	}

	t.Logf("FetchAndRenderChart failed as expected. errors was: %s", err.Error())
}

// TestRendererInvalidDeploymentName tests if the renderer returns an error
// when the chart renders a deployment that doesn't have a name templated with
// the release's name.
func TestRendererInvalidDeploymentName(t *testing.T) {
	it := buildInstallationTarget(
		shippertesting.TestNamespace,
		shippertesting.TestApp,
		buildChart("reviews-api", "invalid-deployment-name", repoUrl))

	_, err := FetchAndRenderChart(localFetchChart, it)

	if err == nil {
		t.Fatal("FetchAndRenderChart should fail, invalid deployment name")
	}

	if _, ok := err.(shippererrors.InvalidChartError); !ok {
		t.Fatalf("FetchAndRenderChart should fail with InvalidChartError, got %v instead", err)
	}

	t.Logf("FetchAndRenderChart failed as expected. errors was: %s", err.Error())
}

// TestRendererMultiServiceNoLB tests if the renderer returns an error when the
// chart renders multiple services, but none with LBLabel to denote the one
// that Shipper should use for traffic shifting.
func TestRendererMultiServiceNoLB(t *testing.T) {
	it := buildInstallationTarget(
		shippertesting.TestNamespace,
		shippertesting.TestApp,
		buildChart("reviews-api", "multi-service-no-lb", repoUrl))

	_, err := FetchAndRenderChart(localFetchChart, it)

	if err == nil {
		t.Fatal("FetchAndRenderChart should fail, chart has multiple services but none with LBLabel")
	}

	expected := fmt.Sprintf(
		`one and only one v1.Service object with label %q is required, but 0 found instead`,
		shipper.LBLabel)

	if err.Error() != expected {
		t.Fatalf(
			"FetchAndRenderChart should fail with %q, got different error instead: %s",
			expected, err)
	}
}

func TestInstallerServiceWithReleaseNoWorkaround(t *testing.T) {
	it := buildInstallationTarget(
		shippertesting.TestNamespace,
		shippertesting.TestApp,
		buildChart("reviews-api", "0.0.1", repoUrl))

	// Disabling the helm workaround
	delete(it.ObjectMeta.Labels, shipper.HelmWorkaroundLabel)

	_, err := FetchAndRenderChart(localFetchChart, it)

	if err == nil {
		t.Fatal("Expected error, none raised")
	}
	if matched, regexErr := regexp.MatchString("This will break shipper traffic shifting logic", err.Error()); regexErr != nil {
		t.Fatalf("Failed to match the error message against the regex: %s", regexErr)
	} else if !matched {
		t.Fatalf("Unexpected error: %s", err)
	}
}

/*
func TestInstallerSingleServiceNoLB(t *testing.T) {
	cluster := buildCluster("minikube-a")
	appName := "reviews-api"
	shippertesting.TestNamespace := "reviews-api"

	chart := buildChart(appName, "single-service-no-lb", repoUrl)

	it := buildInstallationTarget(shippertesting.TestNamespace, appName, []string{cluster.Name}, chart)
	configMapAnchor := anchor.CreateConfigMapAnchor(it)
	installer, err := newInstaller(it)
	if err != nil {
		t.Fatalf("could not initialize the installer: %s", err)
	}
	svc := loadService("baseline")
	svc.SetOwnerReferences(append(svc.GetOwnerReferences(), anchor.ConfigMapAnchorToOwnerReference(configMapAnchor)))

	f := newFixture(objectsPerClusterMap{cluster.Name: nil})
	fakeCluster := f.Clusters[cluster.Name]

	expectedDynamicActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, nil),
	}

	expectedActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-anchor"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, nil),
		shippertesting.NewDiscoveryAction("services"),
		shippertesting.NewDiscoveryAction("deployments"),
	}

	if err := installer.install(f.KubeClient, restConfig, f.DynamicClientBuilder); err != nil {
		t.Fatal(err)
	}

	shippertesting.ShallowCheckActions(expectedDynamicActions, fakeCluster.DynamicClient.Actions(), t)
	shippertesting.ShallowCheckActions(expectedActions, f.KubeClient.Actions(), t)

	filteredActions := filterActions(f.KubeClient.Actions(), "create")
	filteredActions = append(filteredActions, filterActions(fakeCluster.DynamicClient.Actions(), "create")...)

	validateAction(t, filteredActions[0], "ConfigMap")
	validateServiceCreateAction(t, svc, validateAction(t, filteredActions[1], "Service"))
	validateDeploymentCreateAction(t, validateAction(t, filteredActions[2], "Deployment"), map[string]string{"app": "reviews-api"})
}

func TestInstallerSingleServiceWithLB(t *testing.T) {
	cluster := buildCluster("minikube-a")
	appName := "reviews-api"
	shippertesting.TestNamespace := "reviews-api"

	chart := buildChart(appName, "single-service-with-lb", repoUrl)

	it := buildInstallationTarget(shippertesting.TestNamespace, appName, []string{cluster.Name}, chart)
	configMapAnchor := anchor.CreateConfigMapAnchor(it)
	installer, err := newInstaller(it)
	if err != nil {
		t.Fatalf("could not initialize the installer: %s", err)
	}
	svc := loadService("baseline")
	svc.SetOwnerReferences(append(svc.GetOwnerReferences(), anchor.ConfigMapAnchorToOwnerReference(configMapAnchor)))

	f := newFixture(objectsPerClusterMap{cluster.Name: nil})
	fakeCluster := f.Clusters[cluster.Name]

	expectedDynamicActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, nil),
	}

	expectedActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-anchor"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, nil),
		shippertesting.NewDiscoveryAction("services"),
		shippertesting.NewDiscoveryAction("deployments"),
	}

	if err := installer.install(f.KubeClient, restConfig, f.DynamicClientBuilder); err != nil {
		t.Fatal(err)
	}

	shippertesting.ShallowCheckActions(expectedDynamicActions, fakeCluster.DynamicClient.Actions(), t)
	shippertesting.ShallowCheckActions(expectedActions, f.KubeClient.Actions(), t)

	filteredActions := filterActions(f.KubeClient.Actions(), "create")
	filteredActions = append(filteredActions, filterActions(fakeCluster.DynamicClient.Actions(), "create")...)

	validateAction(t, filteredActions[0], "ConfigMap")
	validateServiceCreateAction(t, svc, validateAction(t, filteredActions[1], "Service"))
	validateDeploymentCreateAction(t, validateAction(t, filteredActions[2], "Deployment"), map[string]string{"app": "reviews-api"})
}

func TestInstallerMultiServiceWithLB(t *testing.T) {
	cluster := buildCluster("minikube-a")
	appName := "reviews-api"
	shippertesting.TestNamespace := "reviews-api"

	chart := buildChart(appName, "multi-service-with-lb", repoUrl)

	it := buildInstallationTarget(shippertesting.TestNamespace, appName, []string{cluster.Name}, chart)
	configMapAnchor := anchor.CreateConfigMapAnchor(it)
	installer, err := newInstaller(it)
	if err != nil {
		t.Fatalf("could not initialize the installer: %s", err)
	}
	svc := loadService("baseline")
	svc.SetOwnerReferences(append(svc.GetOwnerReferences(), anchor.ConfigMapAnchorToOwnerReference(configMapAnchor)))

	f := newFixture(objectsPerClusterMap{cluster.Name: nil})
	fakeCluster := f.Clusters[cluster.Name]

	expectedDynamicActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-reviews-api-staging"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, "0.0.1-reviews-api"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, nil),
	}

	expectedActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, "0.0.1-anchor"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, nil),
		shippertesting.NewDiscoveryAction("services"),
		shippertesting.NewDiscoveryAction("deployments"),
	}

	if err := installer.install(f.KubeClient, restConfig, f.DynamicClientBuilder); err != nil {
		t.Fatal(err)
	}

	shippertesting.ShallowCheckActions(expectedDynamicActions, fakeCluster.DynamicClient.Actions(), t)
	shippertesting.ShallowCheckActions(expectedActions, f.KubeClient.Actions(), t)

	filteredActions := filterActions(f.KubeClient.Actions(), "create")
	filteredActions = append(filteredActions, filterActions(fakeCluster.DynamicClient.Actions(), "create")...)

	validateAction(t, filteredActions[0], "ConfigMap")
	validateServiceCreateAction(t, svc, validateAction(t, filteredActions[1], "Service"))
	validateDeploymentCreateAction(t, validateAction(t, filteredActions[3], "Deployment"), map[string]string{"app": "reviews-api"})
}

func TestInstallerMultiServiceWithLBOffTheShelf(t *testing.T) {
	cluster := buildCluster("minikube-a")
	appName := "nginx"
	shippertesting.TestNamespace := "nginx"

	chart := buildChart(appName, "0.1.0", repoUrl)

	it := buildInstallationTarget("nginx", "nginx", []string{cluster.Name}, chart)

	configMapAnchor := anchor.CreateConfigMapAnchor(it)
	installer, err := newInstaller(it)
	if err != nil {
		t.Fatalf("could not initialize the installer: %s", err)
	}
	primarySvc := loadService("nginx-primary")
	secondarySvc := loadService("nginx-secondary")
	primarySvc.SetOwnerReferences(append(primarySvc.GetOwnerReferences(), anchor.ConfigMapAnchorToOwnerReference(configMapAnchor)))
	secondarySvc.SetOwnerReferences(append(secondarySvc.GetOwnerReferences(), anchor.ConfigMapAnchorToOwnerReference(configMapAnchor)))

	f := newFixture(objectsPerClusterMap{cluster.Name: nil})
	fakeCluster := f.Clusters[cluster.Name]

	expectedDynamicActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.1.0-nginx"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, "0.1.0-nginx-staging"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "services", Version: "v1"}, shippertesting.TestNamespace, nil),
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, "0.1.0-nginx"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "deployments", Version: "v1", Group: "apps"}, shippertesting.TestNamespace, nil),
	}

	expectedActions := []kubetesting.Action{
		kubetesting.NewGetAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, "0.1.0-anchor"),
		kubetesting.NewCreateAction(schema.GroupVersionResource{Resource: "configmaps", Version: "v1"}, shippertesting.TestNamespace, nil),
		shippertesting.NewDiscoveryAction("services"),
		shippertesting.NewDiscoveryAction("deployments"),
	}

	if err := installer.install(f.KubeClient, restConfig, f.DynamicClientBuilder); err != nil {
		t.Fatal(err)
	}

	shippertesting.ShallowCheckActions(expectedActions, f.KubeClient.Actions(), t)
	shippertesting.ShallowCheckActions(expectedDynamicActions, fakeCluster.DynamicClient.Actions(), t)

	filteredActions := filterActions(f.KubeClient.Actions(), "create")
	filteredActions = append(filteredActions, filterActions(fakeCluster.DynamicClient.Actions(), "create")...)
	validateAction(t, filteredActions[0], "ConfigMap")
	validateServiceCreateAction(t, primarySvc, validateAction(t, filteredActions[1], "Service"))
	validateServiceCreateAction(t, secondarySvc, validateAction(t, filteredActions[2], "Service"))
	validateDeploymentCreateAction(t, validateAction(t, filteredActions[3], "Deployment"), map[string]string{})
}


*/
