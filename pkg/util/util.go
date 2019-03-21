package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/appscode/stash/apis"
	api_v1beta1 "github.com/appscode/stash/apis/stash/v1beta1"
	"github.com/appscode/stash/pkg/restic"
	"github.com/pkg/errors"
	"kmodules.xyz/client-go/meta"
	store "kmodules.xyz/objectstore-api/api/v1"
)

var (
	ServiceName string
)

type RepoLabelData struct {
	WorkloadKind string
	WorkloadName string
	PodName      string
	NodeName     string
}

func GetHostName(target *api_v1beta1.Target) (string, error) {
	if target == nil { // target nil for cluster backup
		return "host-0", nil
	}
	switch target.Ref.Kind {
	case apis.KindStatefulSet:
		podName := os.Getenv("POD_NAME")
		if podName == "" {
			return "", fmt.Errorf("missing podName for %s", apis.KindStatefulSet)
		}
		podInfo := strings.Split(podName, "-")
		podOrdinal := podInfo[len(podInfo)-1]
		return "host-" + podOrdinal, nil
	case apis.KindDaemonSet:
		nodeName := os.Getenv("POD_NAME")
		if nodeName == "" {
			return "", fmt.Errorf("missing nodeName for %s", apis.KindDaemonSet)
		}
		return nodeName, nil
	default:
		return "host-0", nil
	}
}

func PushgatewayURL() string {
	// called by operator, returning its own namespace. Since pushgateway runs as a side-car with operator, this works!
	return fmt.Sprintf("http://%s.%s.svc:56789", ServiceName, meta.Namespace())
}

func BackupModel(kind string) string {
	switch kind {
	case apis.KindDeployment, apis.KindReplicaSet, apis.KindReplicationController, apis.KindStatefulSet, apis.KindDaemonSet:
		return ModelSidecar
	default:
		return ModelCronJob
	}
}

func GetRepoNameAndSnapshotID(snapshotName string) (repoName, snapshotId string, err error) {
	if len(snapshotName) < 9 {
		err = errors.New("invalid snapshot name")
		return
	}
	tokens := strings.SplitN(snapshotName, "-", -1)
	if len(tokens) < 2 {
		err = errors.New("invalid snapshot name")
		return
	}
	snapshotId = tokens[len(tokens)-1]
	if len(snapshotId) != SnapshotIDLength {
		err = errors.New("invalid snapshot name")
		return
	}

	repoName = strings.TrimSuffix(snapshotName, "-"+snapshotId)
	return
}

func FixBackendPrefix(backend *store.Backend, autoPrefix string) *store.Backend {
	if backend.Local != nil {
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, autoPrefix)
		backend.Local.SubPath = strings.TrimSuffix(backend.Local.SubPath, "/")
	} else if backend.S3 != nil {
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, autoPrefix)
		backend.S3.Prefix = strings.TrimSuffix(backend.S3.Prefix, "/")
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket)
		backend.S3.Prefix = strings.TrimPrefix(backend.S3.Prefix, "/")
	} else if backend.GCS != nil {
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, autoPrefix)
		backend.GCS.Prefix = strings.TrimSuffix(backend.GCS.Prefix, "/")
	} else if backend.Azure != nil {
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, autoPrefix)
		backend.Azure.Prefix = strings.TrimSuffix(backend.Azure.Prefix, "/")
	} else if backend.Swift != nil {
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, autoPrefix)
		backend.Swift.Prefix = strings.TrimSuffix(backend.Swift.Prefix, "/")
	} else if backend.B2 != nil {
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, autoPrefix)
		backend.B2.Prefix = strings.TrimSuffix(backend.B2.Prefix, "/")
	}
	return backend
}

func GetEndpoint(backend *store.Backend) string {
	if backend.S3 != nil {
		return backend.S3.Endpoint
	}
	return ""
}

func GetBucketAndPrefix(backend *store.Backend) (string, string, error) {
	if backend.Local != nil {
		return "", filepath.Join(backend.Local.MountPath, strings.TrimPrefix(backend.Local.SubPath, "/")), nil
	} else if backend.S3 != nil {
		return backend.S3.Bucket, strings.TrimPrefix(backend.S3.Prefix, backend.S3.Bucket+"/"), nil
	} else if backend.GCS != nil {
		return backend.GCS.Bucket, backend.GCS.Prefix, nil
	} else if backend.Azure != nil {
		return backend.Azure.Container, backend.Azure.Prefix, nil
	} else if backend.Swift != nil {
		return backend.Swift.Container, backend.Swift.Prefix, nil
	}
	return "", "", errors.New("unknown backend type.")
}

// TODO: use constant / move to store
func GetProvider(backend store.Backend) (string, error) {
	if backend.Local != nil {
		return restic.ProviderLocal, nil
	} else if backend.S3 != nil {
		return restic.ProviderS3, nil
	} else if backend.GCS != nil {
		return restic.ProviderGCS, nil
	} else if backend.Azure != nil {
		return restic.ProviderAzure, nil
	} else if backend.Swift != nil {
		return restic.ProviderSwift, nil
	} else if backend.B2 != nil {
		return restic.ProviderB2, nil
	}
	return "", errors.New("unknown backend type.")
}

func ExtractDataFromRepositoryLabel(labels map[string]string) (data RepoLabelData, err error) {
	var ok bool
	data.WorkloadKind, ok = labels["workload-kind"]
	if !ok {
		return data, errors.New("workload-kind not found in repository labels")
	}

	data.WorkloadName, ok = labels["workload-name"]
	if !ok {
		return data, errors.New("workload-name not found in repository labels")
	}

	data.PodName, ok = labels["pod-name"]
	if !ok {
		data.PodName = ""
	}

	data.NodeName, ok = labels["node-name"]
	if !ok {
		data.NodeName = ""
	}
	return data, nil
}
