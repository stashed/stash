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

package name

import (
	"fmt"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/crane"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
)

type Image struct {
	Original   string
	Name       string
	Registry   string
	Repository string
	Tag        string
	Digest     string
}

func parseReference(s string, extractTag bool, opts ...name.Option) (*Image, error) {
	ref, err := name.ParseReference(s, opts...)
	if err != nil {
		return nil, err
	}

	var img Image
	switch u := ref.(type) {
	case name.Tag:
		img.Registry = u.RegistryStr()
		img.Repository = u.RepositoryStr()
		img.Tag = u.TagStr()
		if extractTag {
			img.Digest, err = crane.Digest(s, crane.WithAuthFromKeychain(authn.DefaultKeychain))
			if err != nil {
				return nil, err
			}
		} else {
			img.Digest = ""
		}
		img.Name = u.Name()
		img.Original = u.String()
	case name.Digest:
		img.Registry = u.RegistryStr()
		img.Repository = u.RepositoryStr()
		if tag, err := name.NewTag(
			strings.Split(s, "@")[0],                 // skip the digest
			append(opts, name.WithDefaultTag(""))..., // don't use defaultTag "latest"
		); err == nil {
			img.Tag = tag.TagStr()
		}
		img.Digest = u.DigestStr()
		img.Name = u.Name()
		img.Original = u.String()
	default:
		return nil, fmt.Errorf("unknown image %T", ref)
	}
	return &img, nil
}

func ParseReference(s string, opts ...name.Option) (*Image, error) {
	return parseReference(s, false, opts...)
}

func ParseOrExtractDigest(s string, opts ...name.Option) (*Image, error) {
	return parseReference(s, true, opts...)
}

func IsPrivateImage(img string) (bool, error) {
	reference, err := name.ParseReference(img)
	if err != nil {
		return false, err
	}

	_, err = remote.Get(reference, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err == nil {
		return false, nil
	}
	if strings.Contains(err.Error(), "UNAUTHORIZED") {
		return true, nil
	}
	if strings.Contains(err.Error(), "MANIFEST_UNKNOWN") { // If the image is kind loaded (not available online)
		return true, nil
	}

	return true, err
}
