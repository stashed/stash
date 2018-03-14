package cli

import (
	"testing"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"strings"
	"github.com/ghodss/yaml"
	"os"
)

const restic = `
apiVersion: stash.appscode.com/v1alpha1
kind: Restic
metadata:
  name: stash-gitlab
  namespace: default
spec:
  backend:
    s3:
      bucket: backups-gitlab
      endpoint: http://minio-0.s3.svc:9000
      prefix: gitlab-data
    storageSecretName: stash-s3-0
  fileGroups:
  - path: /home/git/data
    retentionPolicyName: keep-3h-7d-8w-12m
  retentionPolicies:
  - keepDaily: 7
    keepHourly: 3
    keepMonthly: 12
    keepWeekly: 8
    name: keep-3h-7d-8w-12m
    prune: true
  schedule: '@every 6h'
  selector:
    matchLabels:
      app: gitlab
  volumeMounts:
  - mountPath: /home/git/data
    name: gitlab-persistent-storage
`

func TestBackup(t *testing.T) {
	var resource api.Restic
	err := yaml.Unmarshal([]byte(strings.TrimSpace(restic)), &resource)
	if err != nil {
		t.Fatal(err)
	}

	wrapper := New(os.TempDir(), true, "foo")

	for _, fg := range resource.Spec.FileGroups {
		err := wrapper.Forget(&resource, fg)
		if err != nil {
			t.Error(err)
		}
	}
}
