package client

import (
	aci "github.com/appscode/restik/api"
	"k8s.io/kubernetes/pkg/api"
	rest "k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"
)

type BackupNamespacer interface {
	Backups(namespace string) BackupInterface
}

const (
	ResourceKindBackup = "Backup"
	ResourceNameBackup = "backup"
	ResourceTypeBackup = "backups"
)

const GroupName = "appscode.com"

type BackupInterface interface {
	List(opts api.ListOptions) (*aci.BackupList, error)
	Get(name string) (*aci.Backup, error)
	Create(backup *aci.Backup) (*aci.Backup, error)
	Update(backup *aci.Backup) (*aci.Backup, error)
	Delete(name string, options *api.DeleteOptions) error
	Watch(opts api.ListOptions) (watch.Interface, error)
	UpdateStatus(backup *aci.Backup) (*aci.Backup, error)
}

type BackupImpl struct {
	r  rest.Interface
	ns string
}

func newBackup(c *ExtensionsClient, namespace string) *BackupImpl {
	return &BackupImpl{c.restClient, namespace}
}

func (c *BackupImpl) List(opts api.ListOptions) (result *aci.BackupList, err error) {
	result = &aci.BackupList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		VersionedParams(&opts, ExtendedCodec).
		Do().
		Into(result)
	return
}

func (c *BackupImpl) Get(name string) (result *aci.Backup, err error) {
	result = &aci.Backup{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		Name(name).
		Do().
		Into(result)
	return
}

func (c *BackupImpl) Create(backup *aci.Backup) (result *aci.Backup, err error) {
	result = &aci.Backup{}
	err = c.r.Post().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		Body(backup).
		Do().
		Into(result)
	return
}

func (c *BackupImpl) Update(backup *aci.Backup) (result *aci.Backup, err error) {
	result = &aci.Backup{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		Name(backup.Name).
		Body(backup).
		Do().
		Into(result)
	return
}

func (c *BackupImpl) Delete(name string, options *api.DeleteOptions) (err error) {
	return c.r.Delete().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		Name(name).
		Body(options).
		Do().
		Error()
}

func (c *BackupImpl) Watch(opts api.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		VersionedParams(&opts, ExtendedCodec).
		Watch()
}

func (c *BackupImpl) UpdateStatus(backup *aci.Backup) (result *aci.Backup, err error) {
	result = &aci.Backup{}
	err = c.r.Put().
		Namespace(c.ns).
		Resource(ResourceTypeBackup).
		Name(backup.Name).
		SubResource("status").
		Body(backup).
		Do().
		Into(result)
	return
}
