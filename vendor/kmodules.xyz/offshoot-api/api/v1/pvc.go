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

package v1

import (
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ObjectMeta is metadata that all persisted resources must have, which includes all objects
// users must create.
type PartialObjectMeta struct {
	// Name must be unique within a namespace. Is required when creating resources, although
	// some resources may allow a client to request the generation of an appropriate name
	// automatically. Name is primarily intended for creation idempotence and configuration
	// definition.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/identifiers#names
	// +optional
	Name string `json:"name,omitempty"`

	// GenerateName is an optional prefix, used by the server, to generate a unique
	// name ONLY IF the Name field has not been provided.
	// If this field is used, the name returned to the client will be different
	// than the name passed. This value will also be combined with a unique suffix.
	// The provided value has the same validation rules as the Name field,
	// and may be truncated by the length of the suffix required to make the value
	// unique on the server.
	//
	// If this field is specified and the generated name exists, the server will
	// NOT return a 409 - instead, it will either return 201 Created or 500 with Reason
	// ServerTimeout indicating a unique name could not be found in the time allotted, and the client
	// should retry (optionally after the time indicated in the Retry-After header).
	//
	// Applied only if Name is not specified.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#idempotency
	// +optional
	GenerateName string `json:"generateName,omitempty"`

	// Namespace defines the space within each name must be unique. An empty namespace is
	// equivalent to the "default" namespace, but "default" is the canonical representation.
	// Not all objects are required to be scoped to a namespace - the value of this field for
	// those objects will be empty.
	//
	// Must be a DNS_LABEL.
	// Cannot be updated.
	// More info: http://kubernetes.io/docs/user-guide/namespaces
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// List of objects depended by this object. If ALL objects in the list have
	// been deleted, this object will be garbage collected. If this object is managed by a controller,
	// then an entry in this list will point to this controller, with the controller field set to true.
	// There cannot be more than one managing controller.
	// +optional
	// +patchMergeKey=uid
	// +patchStrategy=merge
	OwnerReferences []metav1.OwnerReference `json:"ownerReferences,omitempty" patchStrategy:"merge" patchMergeKey:"uid"`
}

// PersistentVolumeClaim is a user's request for and claim to a persistent volume
type PersistentVolumeClaim struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	PartialObjectMeta `json:"metadata,omitempty"`

	// Spec defines the desired characteristics of a volume requested by a pod author.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	Spec core.PersistentVolumeClaimSpec `json:"spec,omitempty"`

	// Status represents the current information/status of a persistent volume claim.
	// Read-only.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	Status core.PersistentVolumeClaimStatus `json:"status,omitempty"`
}

func (in *PersistentVolumeClaim) ToAPIObject() *core.PersistentVolumeClaim {
	if in == nil {
		return nil
	}
	return &core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:            in.Name,
			GenerateName:    in.GenerateName,
			Namespace:       in.Namespace,
			Labels:          in.Labels,
			Annotations:     in.Annotations,
			OwnerReferences: in.OwnerReferences,
		},
		Spec:   in.Spec,
		Status: in.Status,
	}
}

// PersistentVolumeClaimTemplate is used to produce
// PersistentVolumeClaim objects as part of an EphemeralVolumeSource.
type PersistentVolumeClaimTemplate struct {
	// May contain labels and annotations that will be copied into the PVC
	// when creating it. No other fields are allowed and will be rejected during
	// validation.
	//
	// +optional
	PartialObjectMeta `json:"metadata,omitempty"`

	// The specification for the PersistentVolumeClaim. The entire content is
	// copied unchanged into the PVC that gets created from this
	// template. The same fields as in a PersistentVolumeClaim
	// are also valid here.
	Spec core.PersistentVolumeClaimSpec `json:"spec"`
}

func (in *PersistentVolumeClaimTemplate) ToAPIObject() *core.PersistentVolumeClaimTemplate {
	if in == nil {
		return nil
	}
	return &core.PersistentVolumeClaimTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:            in.Name,
			GenerateName:    in.GenerateName,
			Namespace:       in.Namespace,
			Labels:          in.Labels,
			Annotations:     in.Annotations,
			OwnerReferences: in.OwnerReferences,
		},
		Spec: in.Spec,
	}
}

// Volume represents a named volume in a pod that may be accessed by any container in the pod.
type Volume struct {
	// name of the volume.
	// Must be a DNS_LABEL and unique within the pod.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
	Name string `json:"name"`
	// volumeSource represents the location and type of the mounted volume.
	// If not specified, the Volume is implied to be an EmptyDir.
	// This implied behavior is deprecated and will be removed in a future version.
	VolumeSource `json:",inline"`
}

func (in *Volume) ToAPIObject() *core.Volume {
	if in == nil {
		return nil
	}
	return &core.Volume{
		Name:         in.Name,
		VolumeSource: *in.VolumeSource.ToAPIObject(),
	}
}

// Represents the source of a volume to mount.
// Only one of its members may be specified.
type VolumeSource struct {
	// hostPath represents a pre-existing file or directory on the host
	// machine that is directly exposed to the container. This is generally
	// used for system agents or other privileged things that are allowed
	// to see the host machine. Most containers will NOT need this.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#hostpath
	// ---
	// TODO(jonesdl) We need to restrict who can use host directory mounts and who can/can not
	// mount host directories as read/write.
	// +optional
	HostPath *core.HostPathVolumeSource `json:"hostPath,omitempty"`
	// emptyDir represents a temporary directory that shares a pod's lifetime.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#emptydir
	// +optional
	EmptyDir *core.EmptyDirVolumeSource `json:"emptyDir,omitempty"`
	// gcePersistentDisk represents a GCE Disk resource that is attached to a
	// kubelet's host machine and then exposed to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#gcepersistentdisk
	// +optional
	GCEPersistentDisk *core.GCEPersistentDiskVolumeSource `json:"gcePersistentDisk,omitempty"`
	// awsElasticBlockStore represents an AWS Disk resource that is attached to a
	// kubelet's host machine and then exposed to the pod.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#awselasticblockstore
	// +optional
	AWSElasticBlockStore *core.AWSElasticBlockStoreVolumeSource `json:"awsElasticBlockStore,omitempty"`
	// secret represents a secret that should populate this volume.
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#secret
	// +optional
	Secret *core.SecretVolumeSource `json:"secret,omitempty"`
	// nfs represents an NFS mount on the host that shares a pod's lifetime
	// More info: https://kubernetes.io/docs/concepts/storage/volumes#nfs
	// +optional
	NFS *core.NFSVolumeSource `json:"nfs,omitempty"`
	// iscsi represents an ISCSI Disk resource that is attached to a
	// kubelet's host machine and then exposed to the pod.
	// More info: https://examples.k8s.io/volumes/iscsi/README.md
	// +optional
	ISCSI *core.ISCSIVolumeSource `json:"iscsi,omitempty"`
	// glusterfs represents a Glusterfs mount on the host that shares a pod's lifetime.
	// More info: https://examples.k8s.io/volumes/glusterfs/README.md
	// +optional
	Glusterfs *core.GlusterfsVolumeSource `json:"glusterfs,omitempty"`
	// persistentVolumeClaimVolumeSource represents a reference to a
	// PersistentVolumeClaim in the same namespace.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#persistentvolumeclaims
	// +optional
	PersistentVolumeClaim *core.PersistentVolumeClaimVolumeSource `json:"persistentVolumeClaim,omitempty"`
	// rbd represents a Rados Block Device mount on the host that shares a pod's lifetime.
	// More info: https://examples.k8s.io/volumes/rbd/README.md
	// +optional
	RBD *core.RBDVolumeSource `json:"rbd,omitempty"`
	// flexVolume represents a generic volume resource that is
	// provisioned/attached using an exec based plugin.
	// +optional
	FlexVolume *core.FlexVolumeSource `json:"flexVolume,omitempty"`
	// cinder represents a cinder volume attached and mounted on kubelets host machine.
	// More info: https://examples.k8s.io/mysql-cinder-pd/README.md
	// +optional
	Cinder *core.CinderVolumeSource `json:"cinder,omitempty"`
	// cephFS represents a Ceph FS mount on the host that shares a pod's lifetime
	// +optional
	CephFS *core.CephFSVolumeSource `json:"cephfs,omitempty"`
	// flocker represents a Flocker volume attached to a kubelet's host machine. This depends on the Flocker control service being running
	// +optional
	Flocker *core.FlockerVolumeSource `json:"flocker,omitempty"`
	// downwardAPI represents downward API about the pod that should populate this volume
	// +optional
	DownwardAPI *core.DownwardAPIVolumeSource `json:"downwardAPI,omitempty"`
	// fc represents a Fibre Channel resource that is attached to a kubelet's host machine and then exposed to the pod.
	// +optional
	FC *core.FCVolumeSource `json:"fc,omitempty"`
	// azureFile represents an Azure File Service mount on the host and bind mount to the pod.
	// +optional
	AzureFile *core.AzureFileVolumeSource `json:"azureFile,omitempty"`
	// configMap represents a configMap that should populate this volume
	// +optional
	ConfigMap *core.ConfigMapVolumeSource `json:"configMap,omitempty"`
	// vsphereVolume represents a vSphere volume attached and mounted on kubelets host machine
	// +optional
	VsphereVolume *core.VsphereVirtualDiskVolumeSource `json:"vsphereVolume,omitempty"`
	// quobyte represents a Quobyte mount on the host that shares a pod's lifetime
	// +optional
	Quobyte *core.QuobyteVolumeSource `json:"quobyte,omitempty"`
	// azureDisk represents an Azure Data Disk mount on the host and bind mount to the pod.
	// +optional
	AzureDisk *core.AzureDiskVolumeSource `json:"azureDisk,omitempty"`
	// photonPersistentDisk represents a PhotonController persistent disk attached and mounted on kubelets host machine
	PhotonPersistentDisk *core.PhotonPersistentDiskVolumeSource `json:"photonPersistentDisk,omitempty"`
	// projected items for all in one resources secrets, configmaps, and downward API
	Projected *core.ProjectedVolumeSource `json:"projected,omitempty"`
	// portworxVolume represents a portworx volume attached and mounted on kubelets host machine
	// +optional
	PortworxVolume *core.PortworxVolumeSource `json:"portworxVolume,omitempty"`
	// scaleIO represents a ScaleIO persistent volume attached and mounted on Kubernetes nodes.
	// +optional
	ScaleIO *core.ScaleIOVolumeSource `json:"scaleIO,omitempty"`
	// storageOS represents a StorageOS volume attached and mounted on Kubernetes nodes.
	// +optional
	StorageOS *core.StorageOSVolumeSource `json:"storageos,omitempty"`
	// csi (Container Storage Interface) represents ephemeral storage that is handled by certain external CSI drivers (Beta feature).
	// +optional
	CSI *core.CSIVolumeSource `json:"csi,omitempty"`
	// ephemeral represents a volume that is handled by a cluster storage driver.
	// The volume's lifecycle is tied to the pod that defines it - it will be created before the pod starts,
	// and deleted when the pod is removed.
	//
	// Use this if:
	// a) the volume is only needed while the pod runs,
	// b) features of normal volumes like restoring from snapshot or capacity
	//    tracking are needed,
	// c) the storage driver is specified through a storage class, and
	// d) the storage driver supports dynamic volume provisioning through
	//    a PersistentVolumeClaim (see EphemeralVolumeSource for more
	//    information on the connection between this volume type
	//    and PersistentVolumeClaim).
	//
	// Use PersistentVolumeClaim or one of the vendor-specific
	// APIs for volumes that persist for longer than the lifecycle
	// of an individual pod.
	//
	// Use CSI for light-weight local ephemeral volumes if the CSI driver is meant to
	// be used that way - see the documentation of the driver for
	// more information.
	//
	// A pod can use both types of ephemeral volumes and
	// persistent volumes at the same time.
	//
	// +optional
	Ephemeral *EphemeralVolumeSource `json:"ephemeral,omitempty"`
}

func (in *VolumeSource) ToAPIObject() *core.VolumeSource {
	if in == nil {
		return nil
	}
	return &core.VolumeSource{
		HostPath:              in.HostPath,
		EmptyDir:              in.EmptyDir,
		GCEPersistentDisk:     in.GCEPersistentDisk,
		AWSElasticBlockStore:  in.AWSElasticBlockStore,
		Secret:                in.Secret,
		NFS:                   in.NFS,
		ISCSI:                 in.ISCSI,
		Glusterfs:             in.Glusterfs,
		PersistentVolumeClaim: in.PersistentVolumeClaim,
		RBD:                   in.RBD,
		FlexVolume:            in.FlexVolume,
		Cinder:                in.Cinder,
		CephFS:                in.CephFS,
		Flocker:               in.Flocker,
		DownwardAPI:           in.DownwardAPI,
		FC:                    in.FC,
		AzureFile:             in.AzureFile,
		ConfigMap:             in.ConfigMap,
		VsphereVolume:         in.VsphereVolume,
		Quobyte:               in.Quobyte,
		AzureDisk:             in.AzureDisk,
		PhotonPersistentDisk:  in.PhotonPersistentDisk,
		Projected:             in.Projected,
		PortworxVolume:        in.PortworxVolume,
		ScaleIO:               in.ScaleIO,
		StorageOS:             in.StorageOS,
		CSI:                   in.CSI,
		Ephemeral:             in.Ephemeral.ToAPIObject(),
	}
}

// Represents an ephemeral volume that is handled by a normal storage driver.
type EphemeralVolumeSource struct {
	// Will be used to create a stand-alone PVC to provision the volume.
	// The pod in which this EphemeralVolumeSource is embedded will be the
	// owner of the PVC, i.e. the PVC will be deleted together with the
	// pod.  The name of the PVC will be `<pod name>-<volume name>` where
	// `<volume name>` is the name from the `PodSpec.Volumes` array
	// entry. Pod validation will reject the pod if the concatenated name
	// is not valid for a PVC (for example, too long).
	//
	// An existing PVC with that name that is not owned by the pod
	// will *not* be used for the pod to avoid using an unrelated
	// volume by mistake. Starting the pod is then blocked until
	// the unrelated PVC is removed. If such a pre-created PVC is
	// meant to be used by the pod, the PVC has to updated with an
	// owner reference to the pod once the pod exists. Normally
	// this should not be necessary, but it may be useful when
	// manually reconstructing a broken cluster.
	//
	// This field is read-only and no changes will be made by Kubernetes
	// to the PVC after it has been created.
	//
	// Required, must not be nil.
	VolumeClaimTemplate *PersistentVolumeClaimTemplate `json:"volumeClaimTemplate,omitempty"`

	// ReadOnly is tombstoned to show why 2 is a reserved protobuf tag.
	// ReadOnly bool `json:"readOnly,omitempty"`
}

func (in *EphemeralVolumeSource) ToAPIObject() *core.EphemeralVolumeSource {
	if in == nil {
		return nil
	}
	return &core.EphemeralVolumeSource{
		VolumeClaimTemplate: in.VolumeClaimTemplate.ToAPIObject(),
	}
}

func ConvertVolumes(in []Volume) []core.Volume {
	out := make([]core.Volume, 0, len(in))
	for _, v := range in {
		out = append(out, *v.ToAPIObject())
	}
	return out
}
