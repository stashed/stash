package kutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
)

const (
	timestampFormat = "20060102T150405"
)

type ItemList struct {
	Items []map[string]interface{} `json:"items,omitempty"`
}

type BackupManager struct {
	cluster  string
	config   *rest.Config
	sanitize bool
}

func NewBackupManager(cluster string, config *rest.Config, sanitize bool) BackupManager {
	return BackupManager{
		cluster:  cluster,
		config:   config,
		sanitize: sanitize,
	}
}

type processorFunc func(relPath string, data []byte) error

func (mgr BackupManager) snapshotPrefix(t time.Time) string {
	if mgr.cluster == "" {
		return "snapshot-" + t.UTC().Format(timestampFormat)
	}
	return mgr.cluster + "-" + t.UTC().Format(timestampFormat)
}

func (mgr BackupManager) BackupToDir(backupDir string) error {
	snapshotDir := mgr.snapshotPrefix(time.Now())
	p := func(relPath string, data []byte) error {
		absPath := filepath.Join(backupDir, snapshotDir, relPath)
		dir := filepath.Dir(absPath)
		err := os.MkdirAll(dir, 0777)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(absPath, data, 0644)
	}
	return mgr.Backup(p)
}

func (mgr BackupManager) BackupToTar(backupDir string) error {
	err := os.MkdirAll(backupDir, 0777)
	if err != nil {
		return err
	}

	t := time.Now()
	prefix := mgr.snapshotPrefix(t)

	file, err := os.Create(filepath.Join(backupDir, prefix+".tar.gz"))
	if err != nil {
		return err
	}
	defer file.Close()
	// set up the gzip writer
	gw := gzip.NewWriter(file)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()

	p := func(relPath string, data []byte) error {
		// now lets create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = relPath
		header.Size = int64(len(data))
		header.Mode = 0666
		header.ModTime = t
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// copy the file data to the tarball
		if _, err := io.Copy(tw, bytes.NewReader(data)); err != nil {
			return err
		}
		return nil
	}
	return mgr.Backup(p)
}

func (mgr BackupManager) Backup(process processorFunc) error {
	// ref: https://github.com/kubernetes/ingress-nginx/blob/0dab51d9eb1e5a9ba3661f351114825ac8bfc1af/pkg/ingress/controller/launch.go#L252
	mgr.config.QPS = 1e6
	mgr.config.Burst = 1e6
	if err := rest.SetKubernetesDefaults(mgr.config); err != nil {
		return err
	}
	mgr.config.ContentConfig = dynamic.ContentConfig()

	disClient, err := discovery.NewDiscoveryClientForConfig(mgr.config)
	if err != nil {
		return err
	}
	resourceLists, err := disClient.ServerPreferredResources()
	if err != nil {
		return err
	}
	resourceListBytes, err := yaml.Marshal(resourceLists)
	if err != nil {
		return err
	}
	err = process("resource_lists.yaml", resourceListBytes)
	if err != nil {
		return err
	}

	for _, list := range resourceLists {
		gv, err := schema.ParseGroupVersion(list.GroupVersion)
		if err != nil {
			return err
		}
		for _, r := range list.APIResources {
			if strings.ContainsRune(r.Name, '/') {
				continue
			}
			glog.V(3).Infof("Taking backup of %s apiVersion:%s kind:%s", list.GroupVersion, r.Name)
			mgr.config.GroupVersion = &gv
			mgr.config.APIPath = "/apis"
			if gv.Group == core.GroupName {
				mgr.config.APIPath = "/api"
			}
			client, err := rest.RESTClientFor(mgr.config)
			if err != nil {
				return err
			}
			request := client.Get().Resource(r.Name).Param("pretty", "true")
			resp, err := request.DoRaw()
			if err != nil {
				glog.Errorln(err)
				continue
			}
			items := &ItemList{}
			err = yaml.Unmarshal(resp, &items)
			if err != nil {
				return err
			}
			for _, item := range items.Items {
				var path string
				item["apiVersion"] = list.GroupVersion
				item["kind"] = r.Kind

				md, ok := item["metadata"]
				if ok {
					path = getPathFromSelfLink(md)
					if mgr.sanitize {
						cleanUpObjectMeta(md)
					}
				}
				if mgr.sanitize {
					if spec, ok := item["spec"].(map[string]interface{}); ok {
						switch r.Kind {
						case "Pod":
							item["spec"], err = cleanUpPodSpec(spec)
							if err != nil {
								return err
							}
						case "StatefulSet", "Deployment", "ReplicaSet", "DaemonSet", "ReplicationController", "Job":
							template, ok := spec["template"].(map[string]interface{})
							if ok {
								podSpec, ok := template["spec"].(map[string]interface{})
								if ok {
									template["spec"], err = cleanUpPodSpec(podSpec)
									if err != nil {
										return err
									}
								}
							}
						}
					}
					delete(item, "status")
				}
				data, err := yaml.Marshal(item)
				if err != nil {
					return err
				}
				err = process(path, data)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func cleanUpObjectMeta(md interface{}) {
	meta, ok := md.(map[string]interface{})
	if !ok {
		return
	}
	delete(meta, "creationTimestamp")
	delete(meta, "resourceVersion")
	delete(meta, "uid")
	delete(meta, "generateName")
	delete(meta, "generation")
	annotation, ok := meta["annotations"]
	if !ok {
		return
	}
	annotations, ok := annotation.(map[string]string)
	if !ok {
		return
	}
	cleanUpDecorators(annotations)
}

func cleanUpDecorators(m map[string]string) {
	delete(m, "controller-uid")
	delete(m, "deployment.kubernetes.io/desired-replicas")
	delete(m, "deployment.kubernetes.io/max-replicas")
	delete(m, "deployment.kubernetes.io/revision")
	delete(m, "pod-template-hash")
	delete(m, "pv.kubernetes.io/bind-completed")
	delete(m, "pv.kubernetes.io/bound-by-controller")
}

func cleanUpPodSpec(in map[string]interface{}) (map[string]interface{}, error) {
	b, err := yaml.Marshal(in)
	if err != nil {
		return nil, err
	}
	spec := &core.PodSpec{}
	err = yaml.Unmarshal(b, spec)
	if err != nil {
		return in, nil // Not a podSpec
	}
	spec.DNSPolicy = core.DNSPolicy("")
	spec.NodeName = ""
	if spec.ServiceAccountName == "default" {
		spec.ServiceAccountName = ""
	}
	spec.TerminationGracePeriodSeconds = nil
	for i, c := range spec.Containers {
		c.TerminationMessagePath = ""
		spec.Containers[i] = c
	}
	for i, c := range spec.InitContainers {
		c.TerminationMessagePath = ""
		spec.InitContainers[i] = c
	}
	b, err = yaml.Marshal(spec)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	err = yaml.Unmarshal(b, &out)
	return out, err
}

func getPathFromSelfLink(md interface{}) string {
	meta, ok := md.(map[string]interface{})
	if ok {
		return meta["selfLink"].(string) + ".yaml"
	}
	return ""
}
