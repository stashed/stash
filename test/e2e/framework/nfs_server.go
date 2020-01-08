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

	"github.com/appscode/go/types"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
)

const (
	NFSServer  = "nfs-server"
	NFSService = "nfs-service"
)

var (
	nfsDeploy *apps.Deployment
	nfsSVC    *core.Service
	nfsErr    error
)

func (fi *Invocation) CreateNFSServer() (string, error) {
	// We will deploy NFS server using a Deployment. We will configure our NFS server to store data
	//creating deployment for nfs server
	deploy := fi.DeploymentForNFSServer()

	nfsDeploy, nfsErr = fi.CreateDeploymentForNFSServer(deploy)
	if nfsErr != nil {
		return "", nfsErr
	}
	//fi.AppendToCleanupList(nfsDeploy)

	//creating service for NFS server
	svc := fi.ServiceForNFSServer()
	nfsSVC, nfsErr = fi.CreateServiceForNFSServer(svc)
	if nfsErr != nil {
		return "", nfsErr
	}
	//fi.AppendToCleanupList(nfsSVC)

	nfsErr = apps_util.WaitUntilDeploymentReady(fi.KubeClient, nfsDeploy.ObjectMeta)
	if nfsErr != nil {
		return "", nfsErr
	}
	return nfsSVC.Spec.ClusterIP, nil
}

func (fi *Invocation) DeploymentForNFSServer() apps.Deployment {
	labels := map[string]string{
		"app": NFSServer + fi.App(),
	}
	priviledged := true

	return apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", NFSServer, fi.App()),
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
					// nfs service will select this pod using this label.
					Labels: labels,
				},
				Spec: core.PodSpec{
					// run this containers in NFS server pod
					Containers: []core.Container{
						{
							Name:  "nfs-server",
							Image: "k8s.gcr.io/volume-nfs:latest",
							SecurityContext: &core.SecurityContext{
								Privileged: &priviledged,
								RunAsUser:  types.Int64P(int64(0)),
								RunAsGroup: types.Int64P(int64(0)),
							},
							Args: []string{
								"/exports",
							},
						},
					},
				},
			},
		},
	}
}

func (fi *Invocation) CreateDeploymentForNFSServer(obj apps.Deployment) (*apps.Deployment, error) {
	return fi.KubeClient.AppsV1().Deployments(obj.Namespace).Create(&obj)
}

func (fi *Invocation) ServiceForNFSServer() core.Service {
	return core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", NFSService, fi.App()),
			Namespace: fi.namespace,
		},
		Spec: core.ServiceSpec{
			Ports: []core.ServicePort{
				{
					Name:     "nfs",
					Port:     int32(2049),
					Protocol: "TCP",
				},
				{
					Name:     "udp",
					Port:     int32(111),
					Protocol: "UDP",
				},
			},
			Selector: map[string]string{
				"app": NFSServer + fi.App(),
			},
		},
	}
}

func (fi *Invocation) CreateServiceForNFSServer(obj core.Service) (*core.Service, error) {
	return fi.KubeClient.CoreV1().Services(obj.Namespace).Create(&obj)
}

func (fi *Invocation) DeleteNFSServer() error {
	if err := fi.DeleteDeploymentForNFSServer(nfsDeploy.ObjectMeta); err != nil {
		return err
	}
	if err := fi.DeleteServiceForNFSServer(nfsSVC.ObjectMeta); err != nil {
		return err
	}
	return nil
}

func (fi *Invocation) DeleteDeploymentForNFSServer(objMeta metav1.ObjectMeta) error {
	err := fi.KubeClient.AppsV1().Deployments(objMeta.Namespace).Delete(objMeta.Name, &metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (fi *Invocation) DeleteServiceForNFSServer(objMeta metav1.ObjectMeta) error {
	err := fi.KubeClient.CoreV1().Services(objMeta.Namespace).Delete(objMeta.Name, &metav1.DeleteOptions{})
	if err != nil && !kerr.IsNotFound(err) {
		return err
	}
	return nil
}

func (fi *Invocation) GetNFSService() string {
	return nfsSVC.Spec.ClusterIP
}
