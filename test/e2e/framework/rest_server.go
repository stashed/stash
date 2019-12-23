/*
Copyright The Stash Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package framework

import (
	"fmt"
	"net"

	"github.com/appscode/go/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"gomodules.xyz/cert"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
)

const (
	REST_PUBLIC_CRT_NAME      = "public.crt"
	REST_PRIVATE_KEY_NAME     = "private.key"
	TEST_REST_SERVER_USERNAME = "myuser"
	TEST_REST_SERVER_PASSWORD = "changeit"

	RestServer       = "rest-server"
	RestServerSecret = "rest-server-secret"
	RestPVCStorage   = "rest-pvc-storage"
	RestService      = "rest-service"
)

func (f *Invocation) CreateRestServer(tls bool, ips []net.IP) (string, error) {
	//creating secret for Rest server
	cred := f.SecretForRestServer(ips)
	rcred, err := f.CreateSecret(cred)
	if err != nil {
		return "", err
	}
	f.AppendToCleanupList(rcred)

	//creating deployment for Rest server
	deploy := f.DeploymentForRestServer()

	if !tls { // if tls not enabled then don't mount secret for cacerts
		deploy.Spec.Template.Spec.Containers = f.RemoveRestSecretVolumeMount(deploy.Spec.Template.Spec.Containers)
		deploy.Spec.Template.Spec.Containers[0].Command = []string{
			"sh",
			"-c",
			`/entrypoint.sh &
			pid=$(echo $!)
			echo -n "changeit" | htpasswd -ics $PASSWORD_FILE myuser
			wait $pid`,
		}
		deploy.Spec.Template.Spec.Volumes = f.RemoveRestSecretVolume(deploy.Spec.Template.Spec.Volumes)
	}
	rdeploy, err := f.CreateDeploymentForRestServer(deploy)
	if err != nil {
		return "", err
	}
	f.AppendToCleanupList(rdeploy)

	//creating pvc for Rest server
	pvc := f.PVCForRestServer()
	rpvc, err := f.CreatePersistentVolumeClaimForRestServer(pvc)
	if err != nil {
		return "", err
	}
	f.AppendToCleanupList(rpvc)

	//creating service for Rest server
	svc := f.ServiceForRestServer()
	rsvc, err := f.CreateServiceForRestServer(svc)
	if err != nil {
		return "", err
	}
	f.AppendToCleanupList(rsvc)

	err = apps_util.WaitUntilDeploymentReady(f.KubeClient, rdeploy.ObjectMeta)
	if err != nil {
		return "", err
	}
	return f.RestServiceAddres(), nil
}

func (f *Invocation) SecretForRestServer(ips []net.IP) core.Secret {
	crt, key, err := f.CertStore.NewServerCertPairBytes(f.RestServerSANs(ips))
	Expect(err).NotTo(HaveOccurred())

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestServerSecret, f.App()),
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			REST_PUBLIC_CRT_NAME:  []byte(string(crt) + "\n" + string(f.CertStore.CACertBytes())),
			REST_PRIVATE_KEY_NAME: key,
		},
	}
}

func (f *Invocation) PVCForRestServer() core.PersistentVolumeClaim {
	return core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestPVCStorage, f.App()),
			Namespace: f.namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceName(core.ResourceStorage): resource.MustParse("2Gi"),
				},
			},
			StorageClassName: types.StringP(StandardStorageClass),
		},
	}
}

func (f *Framework) CreatePersistentVolumeClaimForRestServer(obj core.PersistentVolumeClaim) (*core.PersistentVolumeClaim, error) {
	return f.KubeClient.CoreV1().PersistentVolumeClaims(obj.Namespace).Create(&obj)
}

func (f *Invocation) DeploymentForRestServer() apps.Deployment {
	labels := map[string]string{
		"app": RestServer + f.App(),
	}

	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestServer, f.App()),
			Namespace: f.namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},

			Strategy: apps.DeploymentStrategy{
				Type: apps.RecreateDeploymentStrategyType,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					// rest service will select this pod using this label.
					Labels: labels,
				},
				Spec: core.PodSpec{
					// this volumes will be mounted on rest server container
					Volumes: []core.Volume{
						{
							Name: "rest-storage",
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: fmt.Sprintf("%s-%s", RestPVCStorage, f.App()),
								},
							},
						},
						{
							Name: "rest-certs",
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-%s", RestServerSecret, f.App()),
								},
							},
						},
					},
					// run this containers in Rest server pod
					Containers: []core.Container{
						{
							Name:  "rest-server",
							Image: "restic/rest-server",
							Command: []string{
								"sh",
								"-c",
								`/entrypoint.sh --tls-cert /tls/public.crt --tls-key /tls/private.key &
								 pid=$(echo $!)
								 echo -n "changeit" | htpasswd -ics $PASSWORD_FILE myuser
								 wait $pid`,
							},
							Ports: []core.ContainerPort{
								{
									ContainerPort: int32(8000),
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "rest-storage",
									MountPath: "/data",
								},
								{
									Name:      "rest-certs",
									MountPath: "/tls",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (f *Invocation) RemoveRestSecretVolumeMount(containers []core.Container) []core.Container {
	var resp []core.Container
	for _, c := range containers {
		if c.Name == "rest-server" {
			c.VolumeMounts = core_util.EnsureVolumeMountDeleted(c.VolumeMounts, "rest-certs")
		}
		resp = append(resp, c)
	}
	return resp
}

func (f *Invocation) RemoveRestSecretVolume(volumes []core.Volume) []core.Volume {
	return core_util.EnsureVolumeDeleted(volumes, "rest-certs")
}

func (f *Invocation) CreateDeploymentForRestServer(obj apps.Deployment) (*apps.Deployment, error) {
	return f.KubeClient.AppsV1().Deployments(obj.Namespace).Create(&obj)
}

func (f *Invocation) ServiceForRestServer() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestService, f.App()),
			Namespace: f.namespace,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:       "http-1",
					Port:       int32(8000),
					TargetPort: intstr.FromInt(8000),
					Protocol:   core.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				"app": RestServer + f.App(),
			},
		},
	}
}

func (f *Invocation) CreateServiceForRestServer(obj core.Service) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
}

func (f *Invocation) RestServerSANs(ips []net.IP) cert.AltNames {
	altNames := cert.AltNames{
		DNSNames: []string{f.RestServiceAddres()},
		IPs:      ips,
	}
	return altNames
}

func (f *Invocation) RestServiceAddres() string {
	return fmt.Sprintf("%s-%s.%s.svc", RestService, f.App(), f.namespace)

}

func (f Invocation) CreateBackendSecretForRest() (*core.Secret, error) {
	// Create Storage Secret
	cred := f.SecretForRestBackend(false)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Rest credential")
	}
	By(fmt.Sprintf("Creating Storage Secret for Rest: %s/%s", cred.Namespace, cred.Name))
	createdCred, err := f.CreateSecret(cred)
	f.AppendToCleanupList(&cred)

	return createdCred, err
}
