package resolve

import (
	"encoding/json"
	"fmt"

	cs "github.com/appscode/stash/client/clientset/versioned"
	"gomodules.xyz/envsubst"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	core_util "kmodules.xyz/client-go/core/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	ofst "kmodules.xyz/offshoot-api/api/v1"
)

type TaskResolver struct {
	StashClient     cs.Interface
	TaskName        string
	Inputs          map[string]string
	RuntimeSettings ofst.RuntimeSettings
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
			Name:            fmt.Sprintf("%s-%d", function.Name, i), // TODO
			Image:           function.Spec.Image,
			Command:         function.Spec.Command,
			Args:            function.Spec.Args,
			WorkingDir:      function.Spec.WorkingDir,
			Ports:           function.Spec.Ports,
			EnvFrom:         function.Spec.EnvFrom,
			Env:             function.Spec.Env,
			VolumeMounts:    function.Spec.VolumeMounts,
			VolumeDevices:   function.Spec.VolumeDevices,
			Resources:       function.Spec.RuntimeSettings.Resources,
			LivenessProbe:   function.Spec.RuntimeSettings.LivenessProbe,
			ReadinessProbe:  function.Spec.RuntimeSettings.ReadinessProbe,
			Lifecycle:       function.Spec.RuntimeSettings.Lifecycle,
			SecurityContext: function.Spec.RuntimeSettings.SecurityContext,
			ImagePullPolicy: core.PullAlways, // TODO
		}

		// apply RuntimeSettings to Container
		if o.RuntimeSettings.Container != nil {
			container = applyContainerRuntimeSettings(container, *o.RuntimeSettings.Container)
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
		podSpec = applyPodRuntimeSettings(podSpec, *o.RuntimeSettings.Pod)
	}
	return podSpec, nil
}

// TODO: move to kmodules
func applyContainerRuntimeSettings(container core.Container, settings ofst.ContainerRuntimeSettings) core.Container {
	if len(settings.Resources.Limits) > 0 {
		container.Resources.Limits = settings.Resources.Limits
	}
	if len(settings.Resources.Limits) > 0 {
		container.Resources.Requests = settings.Resources.Requests
	}
	if settings.LivenessProbe != nil {
		container.LivenessProbe = settings.LivenessProbe
	}
	if settings.ReadinessProbe != nil {
		container.ReadinessProbe = settings.ReadinessProbe
	}
	if settings.Lifecycle != nil {
		container.Lifecycle = settings.Lifecycle
	}
	if settings.SecurityContext != nil {
		container.SecurityContext = settings.SecurityContext
	}
	return container
}

// TODO: move to kmodules
func applyPodRuntimeSettings(podSpec core.PodSpec, settings ofst.PodRuntimeSettings) core.PodSpec {
	if settings.NodeSelector != nil && len(settings.NodeSelector) > 0 {
		podSpec.NodeSelector = settings.NodeSelector
	}
	if settings.ServiceAccountName != "" {
		podSpec.ServiceAccountName = settings.ServiceAccountName
	}
	if settings.AutomountServiceAccountToken != nil {
		podSpec.AutomountServiceAccountToken = settings.AutomountServiceAccountToken
	}
	if settings.NodeName != "" {
		podSpec.NodeName = settings.NodeName
	}
	if settings.SecurityContext != nil {
		podSpec.SecurityContext = settings.SecurityContext
	}
	if len(settings.ImagePullSecrets) > 0 {
		podSpec.ImagePullSecrets = settings.ImagePullSecrets
	}
	if settings.Affinity != nil {
		podSpec.Affinity = settings.Affinity
	}
	if settings.SchedulerName != "" {
		podSpec.SchedulerName = settings.SchedulerName
	}
	if len(settings.Tolerations) > 0 {
		podSpec.Tolerations = settings.Tolerations
	}
	if settings.PriorityClassName != "" {
		podSpec.PriorityClassName = settings.PriorityClassName
	}
	if settings.Priority != nil {
		podSpec.Priority = settings.Priority
	}
	if len(settings.ReadinessGates) > 0 {
		podSpec.ReadinessGates = settings.ReadinessGates
	}
	if settings.RuntimeClassName != nil {
		podSpec.RuntimeClassName = settings.RuntimeClassName
	}
	if settings.EnableServiceLinks != nil {
		podSpec.EnableServiceLinks = settings.EnableServiceLinks
	}
	return podSpec
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
func ResolveBackend(backend *store.Backend, input map[string]string) error {
	return resolveWithInputs(backend, input)
}
