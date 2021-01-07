/*
Copyright AppsCode Inc. and Contributors

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

package portforward

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Tunnel struct {
	Local     int
	Remote    int
	Namespace string
	Resource  string
	Name      string
	Out       io.Writer
	stopChan  chan struct{}
	readyChan chan struct{}
	config    *rest.Config
	client    rest.Interface
}

type TunnelOptions struct {
	Client    rest.Interface
	Config    *rest.Config
	Resource  string
	Name      string
	Namespace string
	Remote    int
}

func NewTunnel(opt TunnelOptions) *Tunnel {
	return &Tunnel{
		config:    opt.Config,
		client:    opt.Client,
		Resource:  opt.Resource,
		Name:      opt.Name,
		Namespace: opt.Namespace,
		Remote:    opt.Remote,
		stopChan:  make(chan struct{}, 1),
		readyChan: make(chan struct{}, 1),
		Out:       ioutil.Discard,
	}
}

func (t *Tunnel) ForwardPort() error {
	k8sClient, err := kubernetes.NewForConfig(t.config)
	if err != nil {
		return err
	}
	pod, err := t.getFirstSelectedPod(k8sClient)
	if err != nil {
		return errors.Wrap(err, "failed to identify any target pod")
	}

	// If the resource kind is "services", then translate remote port into targetPort
	if t.Resource == "services" {
		err := t.translateRemotePort(k8sClient, pod)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to translate remote port: %d into container port", t.Remote))
		}
	}

	u := t.client.Post().
		Resource("pods").
		Namespace(t.Namespace).
		Name(pod.Name).
		SubResource("portforward").URL()

	transport, upgrader, err := spdy.RoundTripperFor(t.config)
	if err != nil {
		return err
	}
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", u)

	local, err := getAvailablePort()
	if err != nil {
		return errors.Errorf("could not find an available port: %s", err)
	}
	t.Local = local

	ports := []string{fmt.Sprintf("%d:%d", t.Local, t.Remote)}

	pf, err := portforward.New(dialer, ports, t.stopChan, t.readyChan, t.Out, t.Out)
	if err != nil {
		return err
	}

	errChan := make(chan error)
	go func() {
		errChan <- pf.ForwardPorts()
	}()

	select {
	case err = <-errChan:
		return errors.Errorf("forwarding ports: %v", err)
	case <-pf.Ready:
		return nil
	}
}

func (t *Tunnel) Close() {
	close(t.stopChan)
}

func getAvailablePort() (int, error) {
	l, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer l.Close()

	_, p, err := net.SplitHostPort(l.Addr().String())
	if err != nil {
		return 0, err
	}
	port, err := strconv.Atoi(p)
	if err != nil {
		return 0, err
	}
	return port, err
}

func (t *Tunnel) getFirstSelectedPod(k8sClient kubernetes.Interface) (*core.Pod, error) {
	var err error
	var podSelector labels.Selector
	var selector *metav1.LabelSelector

	// Extract the selector from the respective resources
	switch t.Resource {
	case "pods":
		// No further processing is necessary. Just return the pod.
		return k8sClient.CoreV1().Pods(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})

	case "deployments":
		obj, err := k8sClient.AppsV1().Deployments(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		selector = obj.Spec.Selector

	case "daemonsets":
		obj, err := k8sClient.AppsV1().DaemonSets(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		selector = obj.Spec.Selector

	case "statefulsets":
		obj, err := k8sClient.AppsV1().StatefulSets(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		selector = obj.Spec.Selector

	case "services":
		obj, err := k8sClient.CoreV1().Services(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		if obj.Spec.Selector == nil || len(obj.Spec.Selector) == 0 {
			return nil, fmt.Errorf("invalid label selector. Error: %v", err)
		}
		podSelector = labels.SelectorFromSet(obj.Spec.Selector)
	default:
		return nil, fmt.Errorf("unknown resource type: %s", t.Resource)
	}

	if selector != nil {
		podSelector, err = metav1.LabelSelectorAsSelector(selector)
		if err != nil {
			return nil, fmt.Errorf("invalid label selector. Error: %v", err)
		}
	}

	// List the pods selected by the selector
	pods, err := k8sClient.CoreV1().Pods(t.Namespace).List(context.TODO(), metav1.ListOptions{LabelSelector: podSelector.String()})
	if err != nil {
		return nil, err
	}

	// Returns the first running pod
	for i := range pods.Items {
		if pods.Items[i].Status.Phase == core.PodRunning {
			return &pods.Items[i], nil
		}
	}
	return nil, fmt.Errorf("no Pod found for %s/%s", t.Resource, t.Name)
}

func (t *Tunnel) translateRemotePort(k8sClient kubernetes.Interface, pod *core.Pod) error {
	svc, err := k8sClient.CoreV1().Services(t.Namespace).Get(context.TODO(), t.Name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	// find the remote port in the service
	var sp *core.ServicePort
	for _, p := range svc.Spec.Ports {
		if p.Port == int32(t.Remote) {
			sp = &p
			break
		}
	}
	if sp == nil {
		return fmt.Errorf("remote port: %d does not exist in Service: %s", t.Remote, svc.Name)
	}

	// find the port in Pod
	for _, c := range pod.Spec.Containers {
		for _, cp := range c.Ports {
			if sp.TargetPort.Type == intstr.String {
				if sp.TargetPort.StrVal == cp.Name {
					t.Remote = int(cp.ContainerPort)
					return nil
				}
			} else {
				if sp.TargetPort.IntVal == cp.ContainerPort || (sp.TargetPort.IntVal == 0 && sp.Port == cp.ContainerPort) {
					t.Remote = int(cp.ContainerPort)
					return nil

				}
			}
		}
	}
	return fmt.Errorf("remote port: %d does not match with any container port of the selected Pod: %s", t.Remote, pod.Name)
}
