/*
Copyright AppsCode Inc. and Contributors

Licensed under the AppsCode Community License 1.0.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://github.com/appscode/licenses/raw/1.0.0/AppsCode-Community-1.0.0.md

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package snapshot

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"stash.appscode.dev/apimachinery/apis/repositories"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/registry/rest"
)

type customTableConvertor struct {
	qualifiedResource schema.GroupResource
}

// NewCustomTableConvertor creates a default convertor for the provided resource.
func NewCustomTableConvertor(resource schema.GroupResource) rest.TableConvertor {
	return customTableConvertor{qualifiedResource: resource}
}

func (c customTableConvertor) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	var table metav1.Table
	fn := func(obj runtime.Object) error {
		snapshot, ok := obj.(*repositories.Snapshot)
		if !ok {
			return errNotAcceptable{resource: c.qualifiedResource}
		}
		snapshotID := []rune(snapshot.UID)
		table.Rows = append(table.Rows, metav1.TableRow{
			Cells: []interface{}{
				snapshot.GetName(),
				string(snapshotID[:8]),
				snapshot.Status.Repository,
				snapshot.Status.Hostname,
				snapshot.GetCreationTimestamp().Time.UTC().Format(time.RFC3339),
			},
			Object: runtime.RawExtension{Object: obj},
		})
		return nil
	}
	switch {
	case meta.IsListType(object):
		if err := meta.EachListItem(object, fn); err != nil {
			return nil, err
		}
	default:
		if err := fn(object); err != nil {
			return nil, err
		}
	}
	if m, err := meta.ListAccessor(object); err == nil {
		table.ResourceVersion = m.GetResourceVersion()
		table.Continue = m.GetContinue()
		table.RemainingItemCount = m.GetRemainingItemCount()
	} else {
		if m, err := meta.CommonAccessor(object); err == nil {
			table.ResourceVersion = m.GetResourceVersion()
		}
	}
	if opt, ok := tableOptions.(*metav1.TableOptions); !ok || !opt.NoHeaders {
		table.ColumnDefinitions = []metav1.TableColumnDefinition{
			{Name: "Name", Type: "string", Format: "name", Description: "Name of the Snapshot"},
			{Name: "ID", Type: "string", Format: "", Description: "Snapshot ID"},
			{Name: "Repository", Type: "string", Format: "repository", Description: "Name of the repository where the Snapshot was backed up"},
			{Name: "Hostname", Type: "string", Format: "hostname", Description: "Name of the host whose data was backed up"},
			{Name: "Created At", Type: "date", Description: "Timestamp when the snapshot was created"},
		}
	}
	return &table, nil
}

// errNotAcceptable indicates the resource doesn't support Table conversion
type errNotAcceptable struct {
	resource schema.GroupResource
}

func (e errNotAcceptable) Error() string {
	return fmt.Sprintf("the resource %s does not support being converted to a Table", e.resource)
}

func (e errNotAcceptable) Status() metav1.Status {
	return metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    http.StatusNotAcceptable,
		Reason:  metav1.StatusReason("NotAcceptable"),
		Message: e.Error(),
	}
}
