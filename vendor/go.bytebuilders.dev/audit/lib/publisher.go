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

package lib

import (
	"context"
	"fmt"
	gosync "sync"
	"time"

	api "go.bytebuilders.dev/audit/api/v1"

	cloudeventssdk "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding/format"
	cloudevents "github.com/cloudevents/sdk-go/v2/event"
	"github.com/nats-io/nats.go"
	"go.bytebuilders.dev/license-verifier/info"
	"gomodules.xyz/sync"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog/v2"
	kmapi "kmodules.xyz/client-go/api/v1"
	"kmodules.xyz/client-go/discovery"
	"kmodules.xyz/client-go/tools/clusterid"
	identityapi "kmodules.xyz/resource-metadata/apis/identity/v1alpha1"
	identitylib "kmodules.xyz/resource-metadata/pkg/identity"
	cachex "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const eventInterval = 1 * time.Hour

// Informer - informer allows you interact with the underlying informer.
type Informer interface {
	// AddEventHandlerWithResyncPeriod adds an event handler to the shared informer using the
	// specified resync period.  Events to a single handler are delivered sequentially, but there is
	// no coordination between different handlers.
	AddEventHandlerWithResyncPeriod(handler cache.ResourceEventHandler, resyncPeriod time.Duration) (cache.ResourceEventHandlerRegistration, error)
}

type EventCreator func(obj client.Object) (*api.Event, error)

type EventPublisher struct {
	once    sync.Once
	connect func() error

	nats        *NatsConfig
	lu          LicenseIDGetter
	mapper      discovery.ResourceMapper
	createEvent EventCreator

	siMutex gosync.Mutex
	si      *identityapi.SiteInfo
}

func NewEventPublisher(
	nats *NatsConfig,
	mapper discovery.ResourceMapper,
	fn EventCreator,
) *EventPublisher {
	p := &EventPublisher{
		mapper:      mapper,
		createEvent: fn,
	}
	p.connect = func() error {
		p.nats = nats
		return nil
	}
	return p
}

func NewResilientEventPublisher(
	fnConnect func() (*NatsConfig, LicenseIDGetter, error),
	mapper discovery.ResourceMapper,
	fnCreateEvent EventCreator,
) *EventPublisher {
	p := &EventPublisher{
		mapper:      mapper,
		createEvent: fnCreateEvent,
	}
	p.connect = func() error {
		var err error
		p.nats, p.lu, err = fnConnect()
		if err != nil {
			klog.V(5).InfoS("failed to connect with event receiver", "error", err)
		}
		return err
	}
	return p
}

func (p *EventPublisher) NatsClient() (*nats.Conn, error) {
	p.once.Do(p.connect)
	if p.nats == nil {
		return nil, fmt.Errorf("not connected to nats")
	}
	return p.nats.Client, nil
}

func (p *EventPublisher) Publish(ev *api.Event, et api.EventType) error {
	event := cloudeventssdk.NewEvent()
	event.SetID(fmt.Sprintf("%s.%d", ev.Resource.GetUID(), ev.Resource.GetGeneration()))
	// /appscode.com/auditor/license_id/feature/info.ProductName/api_group/api_resource/
	// ref: https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#source-1
	event.SetSource(fmt.Sprintf("/%s/auditor/%s/feature/%s/%s/%s", info.ProdDomain, ev.LicenseID, info.ProductName, ev.ResourceID.Group, ev.ResourceID.Name))
	// obj.getUID
	// ref: https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#subject
	event.SetSubject(string(ev.Resource.GetUID()))
	// builders.byte.auditor.{created, updated, deleted}.v1
	// ref: https://github.com/cloudevents/spec/blob/v1.0.1/spec.md#type
	event.SetType(string(et))
	event.SetTime(time.Now().UTC())

	if err := event.SetData(cloudevents.ApplicationJSON, ev); err != nil {
		return err
	}

	data, err := format.JSON.Marshal(&event)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), natsEventPublishTimeout)
	defer cancel()

	for {
		_, err = p.nats.Client.Request(p.nats.Subject, data, natsRequestTimeout)
		if err == nil {
			cancel()
		} else {
			klog.V(5).Infoln(err)
		}

		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				klog.V(5).Infof("failed to send event : %s", string(data))
			} else if ctx.Err() == context.Canceled {
				klog.V(5).Infof("Published event `%s` to channel `%s` and acknowledged", et, p.nats.Subject)
			}
			return nil
		default:
			time.Sleep(time.Microsecond * 100)
		}
	}
}

func (p *EventPublisher) ForGVK(informer Informer, gvk schema.GroupVersionKind) {
	if gvk.Version == "" || gvk.Kind == "" {
		panic(fmt.Sprintf("incomplete GVK; %+v", gvk))
	}

	h := &ResourceEventPublisher{
		p:        p,
		counters: map[kmapi.OID]int32{},
		createEvent: func(obj client.Object) (*api.Event, error) {
			r := obj.DeepCopyObject().(client.Object)
			r.GetObjectKind().SetGroupVersionKind(gvk)
			r.SetManagedFields(nil)

			ev, err := p.createEvent(r)
			if err != nil {
				return nil, err
			}

			p.once.Do(p.connect)
			if p.nats == nil {
				return nil, fmt.Errorf("not connected to nats")
			}
			ev.LicenseID = p.lu.GetLicenseID()

			return ev, nil
		},
	}
	_, _ = informer.AddEventHandlerWithResyncPeriod(h, eventInterval)
}

type funcNodeLister func() ([]*core.Node, error)

func (p *EventPublisher) SetupSiteInfoPublisher(cfg *rest.Config, kc kubernetes.Interface, factory informers.SharedInformerFactory) error {
	nodeInformer := factory.Core().V1().Nodes().Informer()
	nodeLister := factory.Core().V1().Nodes().Lister()

	return p.setupSiteInfoPublisher(cfg, kc, nodeInformer, func() ([]*core.Node, error) {
		return nodeLister.List(labels.Everything())
	})
}

func (p *EventPublisher) SetupSiteInfoPublisherWithManager(mgr manager.Manager) error {
	cfg := mgr.GetConfig()
	kc, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	nodeInformer, err := mgr.GetCache().GetInformer(context.TODO(), &core.Node{})
	if err != nil {
		return err
	}
	return p.setupSiteInfoPublisher(cfg, kc, nodeInformer, func() ([]*core.Node, error) {
		var nodeList core.NodeList
		if err := mgr.GetCache().List(context.TODO(), &nodeList); err != nil {
			return nil, err
		}
		result := make([]*core.Node, len(nodeList.Items))
		for i := range nodeList.Items {
			result[i] = &nodeList.Items[i]
		}
		return result, nil
	})
}

func (p *EventPublisher) setupSiteInfoPublisher(cfg *rest.Config, kc kubernetes.Interface, nodeInformer cachex.Informer, listNodes funcNodeLister) error {
	var err error
	p.si, err = identitylib.GetSiteInfo(cfg, kc, nil, "")
	if err != nil {
		return err
	}
	if p.si.Product == nil {
		p.si.Product = new(identityapi.ProductInfo)
	}

	event := func(_ client.Object) (*api.Event, error) {
		cmeta, err := clusterid.ClusterMetadata(kc)
		if err != nil {
			return nil, err
		}
		nodes, err := listNodes()
		if err != nil {
			return nil, err
		}

		p.siMutex.Lock()
		p.si.Kubernetes.Cluster = cmeta
		identitylib.RefreshNodeStats(p.si, nodes)
		p.siMutex.Unlock()

		p.once.Do(p.connect)
		if p.nats == nil {
			return nil, fmt.Errorf("not connected to nats")
		}

		licenseID := p.lu.GetLicenseID()
		p.si.Product.LicenseID = licenseID
		p.si.Name = fmt.Sprintf("%s.%s", licenseID, p.si.Product.ProductName)
		ev := &api.Event{
			Resource: p.si,
			ResourceID: kmapi.ResourceID{
				Group:   identityapi.SchemeGroupVersion.Group,
				Version: identityapi.SchemeGroupVersion.Version,
				Name:    identityapi.ResourceSiteInfos,
				Kind:    identityapi.ResourceKindSiteInfo,
				Scope:   kmapi.ClusterScoped,
			},
			LicenseID: licenseID,
		}
		return ev, nil
	}
	_, err = nodeInformer.AddEventHandlerWithResyncPeriod(&SiteInfoPublisher{
		p:           p,
		createEvent: event,
	}, eventInterval)
	return err
}

func (p *EventPublisher) SetupWithManagerForKind(ctx context.Context, mgr manager.Manager, gvk schema.GroupVersionKind) error {
	if p == nil {
		return nil
	}
	i, err := mgr.GetCache().GetInformerForKind(ctx, gvk)
	if err != nil {
		return err
	}
	p.ForGVK(i, gvk)
	return nil
}

func (p *EventPublisher) SetupWithManager(ctx context.Context, mgr manager.Manager, obj client.Object) error {
	gvk, err := apiutil.GVKForObject(obj, mgr.GetScheme())
	if err != nil {
		return err
	}
	return p.SetupWithManagerForKind(ctx, mgr, gvk)
}
