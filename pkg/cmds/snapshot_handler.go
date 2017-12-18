package cmds

import (
	"encoding/json"
	"net/http"
	_ "net/http/pprof"

	"github.com/appscode/pat"
	api "github.com/appscode/stash/apis/stash/v1alpha1"
	cs "github.com/appscode/stash/client/typed/stash/v1alpha1"
	"github.com/appscode/stash/pkg/cli"
	core "k8s.io/api/core/v1"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	PathParamNamespace   = ":namespace"
	PathParamName        = ":name"
	QueryParamAutoPrefix = "autoPrefix"
)

type PrometheusExporter struct {
	kubeClient  kubernetes.Interface
	stashClient cs.StashV1alpha1Interface
	scratchDir  string
}

var _ http.Handler = &PrometheusExporter{}

func (e PrometheusExporter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, found := pat.FromContext(r.Context())
	if !found {
		http.Error(w, "Missing parameters", http.StatusBadRequest)
		return
	}
	namespace := params.Get(PathParamNamespace)
	if namespace == "" {
		http.Error(w, "Missing parameter:"+PathParamNamespace, http.StatusBadRequest)
		return
	}
	name := params.Get(PathParamName)
	if name == "" {
		http.Error(w, "Missing parameter:"+PathParamName, http.StatusBadRequest)
		return
	}
	resticCLI := cli.New(e.scratchDir, true, "")

	var resource *api.Restic
	resource, err := e.stashClient.Restics(namespace).Get(name, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if resource.Spec.Backend.StorageSecretName == "" {
		http.Error(w, "Missing repository secret name", http.StatusBadRequest)
		return
	}
	var secret *core.Secret
	secret, err = e.kubeClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
	if kerr.IsNotFound(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = resticCLI.SetupEnv(resource, secret, r.URL.Query().Get(QueryParamAutoPrefix))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	snapshots, err := resticCLI.ListSnapshots()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	js, err := json.Marshal(snapshots)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
