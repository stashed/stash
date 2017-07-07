package cmds

import (
	"encoding/json"
	"net/http"
	_ "net/http/pprof"

	"github.com/appscode/pat"
	sapi "github.com/appscode/stash/api"
	"github.com/appscode/stash/pkg/cli"
	kerr "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiv1 "k8s.io/client-go/pkg/api/v1"
)

const (
	PathParamNamespace   = ":namespace"
	PathParamName        = ":name"
	QueryParamAutoPrefix = "autoPrefix"
)

func ExportSnapshots(w http.ResponseWriter, r *http.Request) {
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
	resticCLI := cli.New(scratchDir)

	var resource *sapi.Restic
	resource, err := stashClient.Restics(namespace).Get(name)
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
	var secret *apiv1.Secret
	secret, err = kubeClient.CoreV1().Secrets(resource.Namespace).Get(resource.Spec.Backend.StorageSecretName, metav1.GetOptions{})
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
