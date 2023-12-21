package v1alpha1

import (
	kmapi "kmodules.xyz/client-go/api/v1"
)

func (ref *ChartSourceRef) SetDefaults() *ChartSourceRef {
	if ref.SourceRef.APIGroup == "" {
		ref.SourceRef.APIGroup = SourceGroupHelmRepository
	}
	if ref.SourceRef.Kind == "" {
		ref.SourceRef.Kind = SourceKindHelmRepository
	} else if ref.SourceRef.Kind == SourceKindLegacy ||
		ref.SourceRef.Kind == SourceKindLocal ||
		ref.SourceRef.Kind == SourceKindEmbed {
		ref.SourceRef.APIGroup = SourceGroupLegacy
	}
	return ref
}

func (ref *ChartSourceFlatRef) FromAPIObject(obj ChartSourceRef) *ChartSourceFlatRef {
	obj.SetDefaults()

	ref.Name = obj.Name
	ref.Version = obj.Version
	ref.SourceAPIGroup = obj.SourceRef.APIGroup
	ref.SourceKind = obj.SourceRef.Kind
	ref.SourceNamespace = obj.SourceRef.Namespace
	ref.SourceName = obj.SourceRef.Name
	return ref
}

func (ref *ChartSourceFlatRef) ToAPIObject() ChartSourceRef {
	obj := ChartSourceRef{
		Name:    ref.Name,
		Version: ref.Version,
		SourceRef: kmapi.TypedObjectReference{
			APIGroup:  ref.SourceAPIGroup,
			Kind:      ref.SourceKind,
			Namespace: ref.SourceNamespace,
			Name:      ref.SourceName,
		},
	}
	obj.SetDefaults()
	return obj
}
