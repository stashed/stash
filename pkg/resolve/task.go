package resolve

import (
	"encoding/json"
	"fmt"
	"strings"

	"gomodules.xyz/envsubst"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
	ofst_util "kmodules.xyz/offshoot-api/util"
	v1beta1_api "stash.appscode.dev/stash/apis/stash/v1beta1"
	cs "stash.appscode.dev/stash/client/clientset/versioned"
	"stash.appscode.dev/stash/pkg/util"
)

type TaskResolver struct {
	StashClient     cs.Interface
	TaskName        string
	Inputs          map[string]string
	RuntimeSettings ofst.RuntimeSettings
	TempDir         v1beta1_api.EmptyDirSettings
}

func (o TaskResolver) GetPodSpec() (core.PodSpec, error) {
	task, err := o.StashClient.StashV1beta1().Tasks().Get(o.TaskName, metav1.GetOptions{})
	if err != nil {
		return core.PodSpec{}, err
	}
	// resolve Task with inputs, modify in place
	if err = resolveWithInputs(task, o.Inputs); err != nil {
		return core.PodSpec{}, err
	}

	var containers []core.Container

	// get Functions for Task
	for i, fn := range task.Spec.Steps {
		function, err := o.StashClient.StashV1beta1().Functions().Get(fn.Name, metav1.GetOptions{})
		if err != nil {
			return core.PodSpec{}, fmt.Errorf("can't get Function %s for Task %s, reason: %s", fn.Name, task.Name, err)
		}

		// inputs from params
		inputs := make(map[string]string)
		for _, param := range fn.Params {
			inputs[param.Name] = param.Value
		}
		// merge/replace backup config inputs
		inputs = core_util.UpsertMap(inputs, o.Inputs)

		// resolve Function with inputs, modify in place
		if err = resolveWithInputs(function, inputs); err != nil {
			return core.PodSpec{}, fmt.Errorf("can't resolve Function %s for Task %s, reason: %s", fn.Name, task.Name, err)
		}

		// init ContainerRuntimeSettings to avoid nil pointer
		if function.Spec.RuntimeSettings == nil {
			function.Spec.RuntimeSettings = &ofst.ContainerRuntimeSettings{}
		}

		// container from function spec
		container := core.Container{
			Name:            fmt.Sprintf("%s-%d", strings.ReplaceAll(function.Name, ".", "-"), i), // TODO
			Image:           function.Spec.Image,
			Command:         function.Spec.Command,
			Args:            function.Spec.Args,
			WorkingDir:      function.Spec.WorkingDir,
			Ports:           function.Spec.Ports,
			EnvFrom:         function.Spec.RuntimeSettings.EnvFrom,
			Env:             function.Spec.RuntimeSettings.Env,
			VolumeMounts:    function.Spec.VolumeMounts,
			VolumeDevices:   function.Spec.VolumeDevices,
			Resources:       function.Spec.RuntimeSettings.Resources,
			LivenessProbe:   function.Spec.RuntimeSettings.LivenessProbe,
			ReadinessProbe:  function.Spec.RuntimeSettings.ReadinessProbe,
			Lifecycle:       function.Spec.RuntimeSettings.Lifecycle,
			SecurityContext: function.Spec.RuntimeSettings.SecurityContext,
			ImagePullPolicy: core.PullIfNotPresent,
		}

		// mount tmp volume
		container.VolumeMounts = util.UpsertTmpVolumeMount(container.VolumeMounts)

		// apply RuntimeSettings to Container
		if o.RuntimeSettings.Container != nil {
			container = ofst_util.ApplyContainerRuntimeSettings(container, *o.RuntimeSettings.Container)
		}

		containers = append(containers, container)
	}
	if len(containers) == 0 {
		return core.PodSpec{}, fmt.Errorf("empty steps/containers for Task %s", task.Name)
	}
	// podSpec from task
	podSpec := core.PodSpec{
		Volumes:        task.Spec.Volumes,
		InitContainers: containers[:len(containers)-1],
		Containers:     containers[len(containers)-1:],
		RestartPolicy:  core.RestartPolicyNever, // TODO: use OnFailure ?
	}
	// apply RuntimeSettings to PodSpec
	if o.RuntimeSettings.Pod != nil {
		podSpec = ofst_util.ApplyPodRuntimeSettings(podSpec, *o.RuntimeSettings.Pod)
	}
	// always upsert tmp volume
	podSpec.Volumes = util.UpsertTmpVolume(podSpec.Volumes, o.TempDir)
	return podSpec, nil
}

func resolveWithInputs(obj interface{}, inputs map[string]string) error {
	// convert to JSON, apply replacements and convert back to struct
	jsonObj, err := json.Marshal(obj)
	if err != nil {
		return err
	}
	resolved, err := envsubst.EvalMap(string(jsonObj), inputs)
	if err != nil {
		return err
	}
	return json.Unmarshal([]byte(resolved), obj)
}

func ResolveBackupTemplate(btpl *v1beta1_api.BackupConfigurationTemplate, input map[string]string) error {
	return resolveWithInputs(btpl, input)
}

func ResolvePVCSpec(pvc *core.PersistentVolumeClaim, input map[string]string) error {
	return resolveWithInputs(pvc, input)
}
