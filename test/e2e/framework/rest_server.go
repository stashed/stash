/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package framework

import (
	"context"
	"fmt"
	"net"

	. "github.com/onsi/gomega"
	"gomodules.xyz/cert"
	"gomodules.xyz/pointer"
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

func (fi *Invocation) CreateRestServer(tls bool, ips []net.IP) (string, error) {
	//creating secret for Rest server
	cred := fi.SecretForRestServer(ips)
	rcred, err := fi.CreateSecret(cred)
	if err != nil {
		return "", err
	}
	fi.AppendToCleanupList(rcred)

	//creating deployment for Rest server
	deploy := fi.DeploymentForRestServer()

	if !tls { // if tls not enabled then don't mount secret for cacerts
		deploy.Spec.Template.Spec.Containers = fi.RemoveRestSecretVolumeMount(deploy.Spec.Template.Spec.Containers)
		deploy.Spec.Template.Spec.Containers[0].Command = []string{
			"sh",
			"-c",
			`/entrypoint.sh &
			pid=$(echo $!)
			echo -n "changeit" | htpasswd -ics $PASSWORD_FILE myuser
			wait $pid`,
		}
		deploy.Spec.Template.Spec.Volumes = fi.RemoveRestSecretVolume(deploy.Spec.Template.Spec.Volumes)
	}
	rdeploy, err := fi.CreateDeploymentForRestServer(deploy)
	if err != nil {
		return "", err
	}
	fi.AppendToCleanupList(rdeploy)

	//creating pvc for Rest server
	pvc := fi.PVCForRestServer()
	rpvc, err := fi.CreatePersistentVolumeClaimForRestServer(pvc)
	if err != nil {
		return "", err
	}
	fi.AppendToCleanupList(rpvc)

	//creating service for Rest server
	svc := fi.ServiceForRestServer()
	rsvc, err := fi.CreateServiceForRestServer(svc)
	if err != nil {
		return "", err
	}
	fi.AppendToCleanupList(rsvc)

	err = apps_util.WaitUntilDeploymentReady(context.TODO(), fi.KubeClient, rdeploy.ObjectMeta)
	if err != nil {
		return "", err
	}
	return fi.RestServiceAddres(), nil
}

func (fi *Invocation) SecretForRestServer(ips []net.IP) core.Secret {
	crt, key, err := fi.CertStore.NewServerCertPairBytes(fi.RestServerSANs(ips))
	Expect(err).NotTo(HaveOccurred())

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestServerSecret, fi.App()),
			Namespace: fi.namespace,
		},
		Data: map[string][]byte{
			REST_PUBLIC_CRT_NAME:  []byte(string(crt) + "\n" + string(fi.CertStore.CACertBytes())),
			REST_PRIVATE_KEY_NAME: key,
		},
	}
}

func (fi *Invocation) PVCForRestServer() core.PersistentVolumeClaim {
	return core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestPVCStorage, fi.App()),
			Namespace: fi.namespace,
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
			StorageClassName: pointer.StringP(StandardStorageClass),
		},
	}
}

func (f *Framework) CreatePersistentVolumeClaimForRestServer(obj core.PersistentVolumeClaim) (*core.PersistentVolumeClaim, error) {
	return f.KubeClient.CoreV1().PersistentVolumeClaims(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (fi *Invocation) DeploymentForRestServer() apps.Deployment {
	labels := map[string]string{
		"app": RestServer + fi.App(),
	}

	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestServer, fi.App()),
			Namespace: fi.namespace,
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
									ClaimName: fmt.Sprintf("%s-%s", RestPVCStorage, fi.App()),
								},
							},
						},
						{
							Name: "rest-certs",
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-%s", RestServerSecret, fi.App()),
								},
							},
						},
					},
					// run this containers in Rest server pod
					Containers: []core.Container{
						{
							Name:  "rest-server",
							Image: "appscodeci/rest-server:latest",
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

func (fi *Invocation) RemoveRestSecretVolumeMount(containers []core.Container) []core.Container {
	var resp []core.Container
	for _, c := range containers {
		if c.Name == "rest-server" {
			c.VolumeMounts = core_util.EnsureVolumeMountDeleted(c.VolumeMounts, "rest-certs")
		}
		resp = append(resp, c)
	}
	return resp
}

func (fi *Invocation) RemoveRestSecretVolume(volumes []core.Volume) []core.Volume {
	return core_util.EnsureVolumeDeleted(volumes, "rest-certs")
}

func (fi *Invocation) CreateDeploymentForRestServer(obj apps.Deployment) (*apps.Deployment, error) {
	return fi.KubeClient.AppsV1().Deployments(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (fi *Invocation) ServiceForRestServer() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", RestService, fi.App()),
			Namespace: fi.namespace,
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
				"app": RestServer + fi.App(),
			},
		},
	}
}

func (fi *Invocation) CreateServiceForRestServer(obj core.Service) (*core.Service, error) {
	return fi.KubeClient.CoreV1().Services(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (fi *Invocation) RestServerSANs(ips []net.IP) cert.AltNames {
	altNames := cert.AltNames{
		DNSNames: []string{fi.RestServiceAddres()},
		IPs:      ips,
	}
	return altNames
}

func (fi *Invocation) RestServiceAddres() string {
	return fmt.Sprintf("%s-%s.%s.svc", RestService, fi.App(), fi.namespace)

}

func (fi *Invocation) CreateRestUser(username string) error {
	// identify the rest-server pod
	pod, err := fi.GetPod(metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-%s", RestServer, fi.App()),
		Namespace: fi.namespace,
	})
	if err != nil {
		return err
	}
	_, _ = fi.ExecOnPod(pod, "create_user", username, TEST_REST_SERVER_PASSWORD)
	return nil
}
