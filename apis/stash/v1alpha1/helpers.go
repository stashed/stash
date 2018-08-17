package v1alpha1

import (
	"hash/fnv"
	"strconv"
	meta_util "github.com/appscode/kutil/meta"
	hashutil "k8s.io/kubernetes/pkg/util/hash"
	"reflect"
	"github.com/golang/glog"
	"github.com/appscode/go/log"
)

func (r Restic) GetSpecHash() string {
	hash := fnv.New64a()
	hashutil.DeepHashObject(hash, r.Spec)
	return strconv.FormatUint(hash.Sum64(), 10)
}

func (e *Restic) AlreadyObserved(other *Restic) bool {
	if e == nil {
		return other == nil
	}
	if other == nil { // && d != nil
		return false
	}
	if e == other {
		return true
	}

	var match bool

	match = meta_util.Equal(e.Spec, other.Spec)
	if match {
		match = reflect.DeepEqual(e.Labels, other.Labels)
	}
	if match {
		match = meta_util.EqualAnnotation(e.Annotations, other.Annotations)
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := meta_util.Diff(other, e)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", meta_util.GetKind(e), e.Namespace, e.Name, diff)
	}
	return match
}

func (e *Recovery) AlreadyObserved(other *Recovery) bool {
	if e == nil {
		return other == nil
	}
	if other == nil { // && d != nil
		return false
	}
	if e == other {
		return true
	}

	var match bool

	if EnableStatusSubresource {
		match = e.Status.ObservedGeneration >= e.Generation
	} else {
		match = meta_util.Equal(e.Spec, other.Spec)
	}
	if match {
		match = reflect.DeepEqual(e.Labels, other.Labels)
	}
	if match {
		match = meta_util.EqualAnnotation(e.Annotations, other.Annotations)
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := meta_util.Diff(other, e)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", meta_util.GetKind(e), e.Namespace, e.Name, diff)
	}
	return match
}


func (e *Repository) AlreadyObserved(other *Repository) bool {
	if e == nil {
		return other == nil
	}
	if other == nil { // && d != nil
		return false
	}
	if e == other {
		return true
	}

	var match bool

	if EnableStatusSubresource {
		match = e.Status.ObservedGeneration >= e.Generation
	} else {
		match = meta_util.Equal(e.Spec, other.Spec)
	}
	if match {
		match = reflect.DeepEqual(e.Labels, other.Labels)
	}
	if match {
		match = meta_util.EqualAnnotation(e.Annotations, other.Annotations)
	}

	if !match && bool(glog.V(log.LevelDebug)) {
		diff := meta_util.Diff(other, e)
		glog.V(log.LevelDebug).Infof("%s %s/%s has changed. Diff: %s", meta_util.GetKind(e), e.Namespace, e.Name, diff)
	}
	return match
}
