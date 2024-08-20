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
	"path/filepath"

	"stash.appscode.dev/apimachinery/apis"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gomodules.xyz/cert"
	"gomodules.xyz/pointer"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	apps_util "kmodules.xyz/client-go/apps/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	meta_util "kmodules.xyz/client-go/meta"
)

const (
	MINIO_PUBLIC_CRT_NAME  = "public.crt"
	MINIO_PRIVATE_KEY_NAME = "private.key"

	MINIO_ACCESS_KEY_ID     = "not@id"
	MINIO_SECRET_ACCESS_KEY = "not@secret"

	MINIO_CERTS_MOUNTPATH = "/root/.minio/certs"
	StandardStorageClass  = "standard"

	MinioServer       = "minio-server"
	MinioServerSecret = "minio-server-secret"
	MinioPVCStorage   = "minio-pvc-storage"
	MinioService      = "minio-service"
	LocalHostIP       = "127.0.0.1"
)

var (
	mcred   core.Secret
	mpvc    core.PersistentVolumeClaim
	mdeploy apps.Deployment
	msrvc   core.Service
)

func (f *Framework) CreateMinioServer(tls bool, ips []net.IP) (string, error) {
	// creating secret for minio server
	mcred = f.SecretForMinioServer(ips)
	_, err := f.CreateSecret(mcred)
	if err != nil {
		return "", err
	}

	// creating pvc for minio server
	mpvc = f.PVCForMinioServer()
	err = f.CreatePersistentVolumeClaimForMinioServer(mpvc)
	if err != nil {
		return "", nil
	}

	// creating deployment for minio server
	mdeploy = f.DeploymentForMinioServer(mpvc, mcred)
	if !tls { // if tls not enabled then don't mount secret for cacerts
		mdeploy.Spec.Template.Spec.Containers = f.RemoveSecretVolumeMount(mdeploy.Spec.Template.Spec.Containers, mcred)
	}

	err = f.CreateDeploymentForMinioServer(mdeploy)
	if err != nil {
		return "", err
	}

	// creating service for minio server
	msrvc = f.ServiceForMinioServer()
	_, err = f.CreateServiceForMinioServer(msrvc)
	if err != nil {
		return "", err
	}
	err = apps_util.WaitUntilDeploymentReady(context.TODO(), f.KubeClient, mdeploy.ObjectMeta)
	if err != nil {
		return "", err
	}
	return f.MinioServiceAddres(), nil
}

func (f *Framework) SecretForMinioServer(ips []net.IP) core.Secret {
	crt, key, err := f.CertStore.NewServerCertPairBytes(f.MinioServerSANs(ips))
	Expect(err).NotTo(HaveOccurred())

	return core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioServerSecret + f.namespace,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			MINIO_PUBLIC_CRT_NAME:  []byte(string(crt) + "\n" + string(f.CertStore.CACertBytes())),
			MINIO_PRIVATE_KEY_NAME: key,
		},
	}
}

func (f *Framework) PVCForMinioServer() core.PersistentVolumeClaim {
	return core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", MinioPVCStorage, f.namespace),
			Namespace: f.namespace,
			Labels: map[string]string{
				// this label will be used to mount this pvc as volume in minio server container
				apis.LabelApp: fmt.Sprintf("%s-%s", MinioServer, f.namespace),
			},
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Resources: core.VolumeResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
			StorageClassName: pointer.StringP(StandardStorageClass),
		},
	}
}

func (f *Framework) CreatePersistentVolumeClaimForMinioServer(obj core.PersistentVolumeClaim) error {
	_, err := f.KubeClient.CoreV1().PersistentVolumeClaims(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) DeploymentForMinioServer(pvc core.PersistentVolumeClaim, secret core.Secret) apps.Deployment {
	labels := map[string]string{
		apis.LabelApp: fmt.Sprintf("%s-%s", MinioServer, f.namespace),
	}

	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      MinioServer,
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
					// minio service will select this pod using this label.
					Labels: labels,
				},
				Spec: core.PodSpec{
					// this volumes will be mounted on minio server container
					Volumes: []core.Volume{
						{
							Name: "minio-storage",
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
						{
							Name: "minio-certs",
							VolumeSource: core.VolumeSource{
								Secret: &core.SecretVolumeSource{
									SecretName: secret.Name,
									Items: []core.KeyToPath{
										{
											Key:  MINIO_PUBLIC_CRT_NAME,
											Path: MINIO_PUBLIC_CRT_NAME,
										},
										{
											Key:  MINIO_PRIVATE_KEY_NAME,
											Path: MINIO_PRIVATE_KEY_NAME,
										},
										{
											Key:  MINIO_PUBLIC_CRT_NAME,
											Path: filepath.Join("CAs", MINIO_PUBLIC_CRT_NAME),
										},
									},
								},
							},
						},
					},
					// run this containers in minio server pod
					Containers: []core.Container{
						{
							Name:            "minio-server",
							Image:           "minio/minio",
							ImagePullPolicy: core.PullIfNotPresent,
							Args: []string{
								"server",
								"--address",
								":443",
								"/storage",
							},
							Env: []core.EnvVar{
								{
									Name:  "MINIO_ACCESS_KEY",
									Value: MINIO_ACCESS_KEY_ID,
								},
								{
									Name:  "MINIO_SECRET_KEY",
									Value: MINIO_SECRET_ACCESS_KEY,
								},
							},
							Ports: []core.ContainerPort{
								{
									ContainerPort: int32(443),
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      "minio-storage",
									MountPath: "/storage",
								},
								{
									Name:      "minio-certs",
									MountPath: MINIO_CERTS_MOUNTPATH,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (f *Framework) RemoveSecretVolumeMount(containers []core.Container, secret core.Secret) []core.Container {
	resp := make([]core.Container, 0)
	for _, c := range containers {
		if c.Name == "minio-server" {
			c.VolumeMounts = core_util.EnsureVolumeMountDeleted(c.VolumeMounts, secret.Name)
		}
		resp = append(resp, c)
	}
	return resp
}

func (f *Framework) CreateDeploymentForMinioServer(obj apps.Deployment) error {
	_, err := f.KubeClient.AppsV1().Deployments(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
	return err
}

func (f *Framework) ServiceForMinioServer() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.GetMinioServiceName(),
			Namespace: f.namespace,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Port:       int32(443),
					TargetPort: intstr.FromInt(443),
					Protocol:   core.ProtocolTCP,
				},
			},
			Selector: map[string]string{
				apis.LabelApp: fmt.Sprintf("%s-%s", MinioServer, f.namespace),
			},
		},
	}
}

func (f *Framework) GetMinioServiceName() string {
	return fmt.Sprintf("%s-%s", MinioService, f.namespace)
}

func (f *Framework) CreateServiceForMinioServer(obj core.Service) (*core.Service, error) {
	return f.KubeClient.CoreV1().Services(obj.Namespace).Create(context.TODO(), &obj, metav1.CreateOptions{})
}

func (f *Framework) DeleteMinioServer() error {
	if err := f.DeleteSecretForMinioServer(mcred.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeletePVCForMinioServer(mpvc.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeleteDeploymentForMinioServer(mdeploy.ObjectMeta); err != nil {
		return err
	}
	if err := f.DeleteServiceForMinioServer(msrvc.ObjectMeta); err != nil {
		return err
	}
	return nil
}

func (f *Framework) DeleteSecretForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Secrets(meta.Namespace).Delete(context.TODO(), meta.Name, *deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeletePVCForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().PersistentVolumeClaims(meta.Namespace).Delete(context.TODO(), meta.Name, *deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeleteDeploymentForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.AppsV1().Deployments(meta.Namespace).Delete(context.TODO(), meta.Name, meta_util.DeleteInBackground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) DeleteServiceForMinioServer(meta metav1.ObjectMeta) error {
	err := f.KubeClient.CoreV1().Services(meta.Namespace).Delete(context.TODO(), meta.Name, *deleteInForeground())
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (f *Framework) MinioServerSANs(ips []net.IP) cert.AltNames {
	altNames := cert.AltNames{
		DNSNames: []string{f.MinioServiceAddres()},
		IPs:      ips,
	}
	return altNames
}

func (f *Framework) MinioServiceAddres() string {
	return fmt.Sprintf("%s.%s.svc", f.GetMinioServiceName(), f.namespace)
}

func (fi *Invocation) CreateBackendSecretForMinio(transformFuncs ...func(in *core.Secret)) (*core.Secret, error) {
	// Create Storage Secret
	cred := fi.SecretForMinioBackend(true)

	if missing, _ := BeZero().Match(cred); missing {
		Skip("Missing Minio credential")
	}

	for _, f := range transformFuncs {
		f(&cred)
	}

	By(fmt.Sprintf("Creating Storage Secret for Minio: %s/%s", cred.Namespace, cred.Name))
	createdCred, err := fi.CreateSecret(cred)
	fi.AppendToCleanupList(createdCred)

	return createdCred, err
}
