package v1alpha1

import (
	"hash/fnv"
	"strconv"

	hashutil "k8s.io/kubernetes/pkg/util/hash"
)

func (r Restic) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}
