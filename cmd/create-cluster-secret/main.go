package main

import (
	"flag"
	"os/user"
	"path"
	"strconv"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	shipper "github.com/bookingcom/shipper/pkg/apis/shipper/v1alpha1"
	"github.com/bookingcom/shipper/pkg/client/clientset/versioned"
)

var (
	masterURL        *string
	kubeconfig       *string
	shipperNamespace *string
	clusterName      *string
	replaceSecret    *bool
	replaceCluster   *bool
)

func init() {
	var _kubeconfig string

	if usr, err := user.Current(); err == nil {
		_kubeconfig = path.Join(usr.HomeDir, ".kube", "config")
	}

	masterURL = flag.String("master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	kubeconfig = flag.String("kubeconfig", _kubeconfig, "Path to a kubeconfig. Only required if out-of-cluster.")
	shipperNamespace = flag.String("shipper-namespace", "shipper-system", "Namespace used by Shipper")
	clusterName = flag.String("cluster-name", "local", "Cluster name that will be used")
	replaceSecret = flag.Bool("replace-secret", false, "Replace existing secret")
	replaceCluster = flag.Bool("replace-cluster", false, "Replace existing Shipper cluster")
}

func main() {
	flag.Parse()

	restCfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubeconfig)
	if err != nil {
		glog.Fatal(err)
	}

	kubeClient := kubernetes.NewForConfigOrDie(restCfg)

	secretData := make(map[string][]byte)
	secretData["tls.ca"] = restCfg.CAData
	secretData["tls.crt"] = restCfg.CertData
	secretData["tls.key"] = restCfg.KeyData

	nsSecrets := kubeClient.CoreV1().Secrets(*shipperNamespace)

	if existingSecret, err := nsSecrets.Get(*clusterName, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			clusterSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name: *clusterName,
					Annotations: map[string]string{
						shipper.SecretChecksumAnnotation:             "some-checksum",
						shipper.SecretClusterSkipTlsVerifyAnnotation: strconv.FormatBool(restCfg.Insecure),
					},
				},
				Type: corev1.SecretTypeOpaque,
				Data: secretData,
			}

			// Only add shipper.SecretClusterSkipTlsVerifyAnnotation if the
			// configuration specifies an insecure connection.
			if restCfg.Insecure {
				clusterSecret.Annotations[shipper.SecretClusterSkipTlsVerifyAnnotation] = strconv.FormatBool(restCfg.Insecure)
			}

			if _, err := kubeClient.CoreV1().Secrets(*shipperNamespace).Create(clusterSecret); err != nil {
				glog.Fatal(err)
			}
			glog.Infof("Successfully created secret for cluster %q", *clusterName)
		} else {
			glog.Fatal(err)
		}
	} else if *replaceSecret {
		existingSecret.Data = secretData
		if existingSecret.Annotations == nil {
			existingSecret.Annotations = map[string]string{}
		}

		// Delete the shipper.SecretClusterSkipTlsVerifyAnnotation if
		// configuration specifies a secure connection, add the annotation
		// it otherwise.
		if !restCfg.Insecure {
			delete(existingSecret.Annotations, shipper.SecretClusterSkipTlsVerifyAnnotation)
		} else {
			existingSecret.Annotations[shipper.SecretClusterSkipTlsVerifyAnnotation] = strconv.FormatBool(restCfg.Insecure)
		}

		if _, err := nsSecrets.Update(existingSecret); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("Successfully replaced secret for cluster %q", *clusterName)
	} else {
		glog.Infof("Nothing to do, secret for cluster %q already exists", *clusterName)
	}

	shipperClient := versioned.NewForConfigOrDie(restCfg)

	if existingCluster, err := shipperClient.ShipperV1alpha1().Clusters().Get(*clusterName, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			cluster := &shipper.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: *clusterName,
				},
				Spec: shipper.ClusterSpec{
					APIMaster:    restCfg.Host,
					Capabilities: []string{},
					Region:       "eu-west",
					Scheduler: shipper.ClusterSchedulerSettings{
						Unschedulable: false,
					},
				},
			}
			if _, err = shipperClient.ShipperV1alpha1().Clusters().Create(cluster); err != nil {
				glog.Fatal(err)
			}
			glog.Infof("Successfully created cluster %q", *clusterName)
		} else {
			glog.Fatal(err)
		}
	} else if *replaceCluster {
		existingCluster.Spec = shipper.ClusterSpec{
			APIMaster:    restCfg.Host,
			Capabilities: []string{},
			Region:       "eu-west",
			Scheduler: shipper.ClusterSchedulerSettings{
				Unschedulable: false,
			},
		}
		if _, err := shipperClient.ShipperV1alpha1().Clusters().Update(existingCluster); err != nil {
			glog.Fatal(err)
		}
		glog.Infof("Successfully replaced cluster %q", *clusterName)
	} else {
		glog.Infof("Nothing to do, cluster %q already exists", *clusterName)
	}
}
