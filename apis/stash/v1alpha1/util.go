package v1alpha1

import (
	"hash/fnv"
	"strconv"

	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

var (
	EnableStatusSubresource bool
)

const (
	ResticKey                = "restic.appscode.com"
	LastAppliedConfiguration = ResticKey + "/last-applied-configuration"
	VersionTag               = ResticKey + "/tag"
	// ResourceVersion will be used to trigger restarts for ReplicaSet and RC pods
	ResourceHash = ResticKey + "/resource-hash"
)

func (b Backup) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, b.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}
