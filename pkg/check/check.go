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

package check

import (
	"context"
	"fmt"

	cs "stash.appscode.dev/apimachinery/client/clientset/versioned/typed/stash/v1alpha1"
	"stash.appscode.dev/stash/pkg/cli"
	"stash.appscode.dev/stash/pkg/eventer"

	"github.com/appscode/go/log"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/reference"
)

const (
	CheckEventComponent = "stash-check"
)

type Options struct {
	Namespace   string
	ResticName  string
	HostName    string
	SmartPrefix string
}

type Controller struct {
	k8sClient   kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	opt         Options
}

func New(k8sClient kubernetes.Interface, stashClient cs.StashV1alpha1Interface, opt Options) *Controller {
	return &Controller{
		k8sClient:   k8sClient,
		stashClient: stashClient,
		opt:         opt,
	}
}

func (c *Controller) Run() (err error) {
	restic, err := c.stashClient.Restics(c.opt.Namespace).Get(context.TODO(), c.opt.ResticName, metav1.GetOptions{})
	if err != nil {
		return
	}

	defer func() {
		if err != nil {
			ref, rerr := reference.GetReference(scheme.Scheme, restic)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					CheckEventComponent,
					ref,
					core.EventTypeWarning,
					eventer.EventReasonFailedToCheck,
					fmt.Sprintf("Check failed for pod %s, reason: %s", c.opt.HostName, err),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", restic.Kind, restic.Name, rerr)
			}
		} else {
			ref, rerr := reference.GetReference(scheme.Scheme, restic)
			if rerr == nil {
				eventer.CreateEventWithLog(
					c.k8sClient,
					CheckEventComponent,
					ref,
					core.EventTypeNormal,
					eventer.EventReasonSuccessfulCheck,
					fmt.Sprintf("Check successful for pod: %s", c.opt.HostName),
				)
			} else {
				log.Errorf("Failed to write event on %s %s. Reason: %s", restic.Kind, restic.Name, rerr)
			}
		}
	}()

	secret, err := c.k8sClient.CoreV1().Secrets(c.opt.Namespace).Get(context.TODO(), restic.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if err != nil {
		return
	}

	cli := cli.New("/tmp", false, c.opt.HostName)
	if _, err = cli.SetupEnv(restic.Spec.Backend, secret, c.opt.SmartPrefix); err != nil {
		return
	}

	err = cli.Check()
	return
}
