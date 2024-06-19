package controller

import (
	"context"
	pipelinev1 "github.com/k-pipe/pipeline-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"strings"
)

// Gets a pipeline job object by name from api server, returns nil,nil if not found
func (r *PipelineJobReconciler) GetJob(ctx context.Context, name types.NamespacedName) (*batchv1.Job, error) {
	res := &batchv1.Job{}
	notexists, err := NotExistsResource(r, ctx, res, name)
	if notexists {
		res = nil
	}
	return res, err
}

/*
create Kubernetes Job
*/
func (r *PipelineJobReconciler) CreateJob(ctx context.Context, log func(string, ...interface{}), pj *pipelinev1.PipelineJob) (*batchv1.Job, error) {
	jobName := pj.Name

	// the labels to be attached to job
	jobLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   jobName,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}
	// the labels to be attached to the pod
	podLabels := map[string]string{
		"app.kubernetes.io/name":       "PipelineSchedule",
		"app.kubernetes.io/instance":   jobName,
		"app.kubernetes.io/version":    "v1",
		"app.kubernetes.io/part-of":    "pipeline-operator",
		"app.kubernetes.io/created-by": "controller-manager", // TODO should we change this?
	}
	var one int32 = 1
	nonIndexed := batchv1.NonIndexedCompletion
	var noSuspend bool = false
	replaceAfterFailed := batchv1.Failed
	var resources corev1.ResourceRequirements
	terminationMessagePath := "/dev/termination-log" // TODO use this

	// variables to collect information about volumes
	volumes := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	initCommands := ""

	// settings working directory
	var sizeInGB int64 = 1
	workdirPath := "/workdir"
	workdirVolumeName := "workdir"
	workdirVolume := corev1.Volume{
		Name: workdirVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{
				Medium:    "",
				SizeLimit: resource.NewQuantity(sizeInGB*(1<<30), resource.BinarySI),
			},
		},
	}
	volumes = append(volumes, workdirVolume)
	volumeMounts = append(volumeMounts, getVolumeMount(workdirVolumeName, workdirPath))
	addInitCommand(&initCommands, "mkdir", "input")

	// collect settings for inputs
	for _, in := range pj.Spec.Inputs {
		if !volumePresentAlready(in.Volume, volumes) {
			volumes = append(volumes, getVolume(in.Volume, true))
			volumeMounts = append(volumeMounts, getVolumeMount(in.Volume, in.MountPath))
			addInitCommand(&initCommands, "ln", "-s", in.MountPath+"/"+in.SourceFile, "/workdir/input/"+in.TargetFile)
		}
	}
	// add output volume for the step
	stepId := pj.Spec.StepId
	volume := jobName // volume and volume claim get same name as job from which the data comes
	volumes = append(volumes, getVolume(volume, false))
	outMountPath := getMountPath(stepId)
	volumeMounts = append(volumeMounts, getVolumeMount(volume, outMountPath))
	addInitCommand(&initCommands, "ln", "-s", outMountPath, "output")
	addInitCommand(&initCommands, "echo", "Initialization", "done")

	// add local workdir volume
	//stepId := pj.Spec.StepId
	//volume := jobName // volume and volume claim get same name as job from which the data comes
	//volumes = append(volumes, getVolume(volume, false))
	//volumeMounts = append(volumeMounts, getVolumeMount(volume, getMountPath(stepId)))

	jobContainer := corev1.Container{
		Name:                     "main",
		Image:                    pj.Spec.JobSpec.Image,
		Command:                  pj.Spec.JobSpec.Command,
		Args:                     pj.Spec.JobSpec.Args,
		WorkingDir:               pj.Spec.JobSpec.WorkingDir,
		Ports:                    []corev1.ContainerPort{},
		EnvFrom:                  []corev1.EnvFromSource{},
		Env:                      []corev1.EnvVar{},
		Resources:                resources,
		ResizePolicy:             []corev1.ContainerResizePolicy{},
		RestartPolicy:            nil, // only for init containers
		VolumeMounts:             volumeMounts,
		VolumeDevices:            []corev1.VolumeDevice{},
		LivenessProbe:            nil,                    // TODO
		ReadinessProbe:           nil,                    // TODO
		StartupProbe:             nil,                    // TODO
		Lifecycle:                nil,                    //
		TerminationMessagePath:   terminationMessagePath, // TODO use this!
		TerminationMessagePolicy: "File",                 // TODO
		ImagePullPolicy:          pj.Spec.JobSpec.ImagePullPolicy,
		SecurityContext:          nil,   // TODO !!!
		Stdin:                    false, // TODO is this security critical?
		StdinOnce:                false,
		TTY:                      false, // TODO is this security critical?
	}

	shellImage := "bash"
	shellCommand := "bash"
	initContainer := corev1.Container{
		Name:            "init",
		Image:           shellImage,
		Command:         []string{shellCommand},
		Args:            []string{"-c", initCommands},
		WorkingDir:      workdirPath,
		VolumeMounts:    volumeMounts,
		ImagePullPolicy: pj.Spec.JobSpec.ImagePullPolicy,
	}
	// define the job object
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: pj.Namespace,
			Labels:    jobLabels,
		},
		Spec: batchv1.JobSpec{
			Parallelism:             &one,
			Completions:             &one,
			ActiveDeadlineSeconds:   pj.Spec.JobSpec.ActiveDeadlineSeconds,
			BackoffLimit:            pj.Spec.JobSpec.BackoffLimit,
			TTLSecondsAfterFinished: pj.Spec.JobSpec.TTLSecondsAfterFinished,
			CompletionMode:          &nonIndexed,
			Suspend:                 &noSuspend,
			PodReplacementPolicy:    &replaceAfterFailed,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					Volumes:                       volumes,
					InitContainers:                []corev1.Container{initContainer},
					Containers:                    []corev1.Container{jobContainer},
					EphemeralContainers:           nil,
					RestartPolicy:                 corev1.RestartPolicyOnFailure,
					TerminationGracePeriodSeconds: pj.Spec.JobSpec.TerminationGracePeriodSeconds,
					// using same TTL on pod level, needed for ResourceQuota reasons, see also https://stackoverflow.com/questions/53506010/kubernetes-difference-between-activedeadlineseconds-in-job-and-pod
					ActiveDeadlineSeconds:        pj.Spec.JobSpec.ActiveDeadlineSeconds,
					DNSPolicy:                    corev1.DNSClusterFirst, // TODO
					DNSConfig:                    nil,                    // TODO
					NodeSelector:                 nil,                    // TODO
					ServiceAccountName:           pj.Spec.JobSpec.ServiceAccountName,
					AutomountServiceAccountToken: nil, // TODO
					HostNetwork:                  false,
					HostPID:                      false,
					HostIPC:                      false,
					ShareProcessNamespace:        nil,                             // TODO
					SecurityContext:              nil,                             // TODO
					ImagePullSecrets:             []corev1.LocalObjectReference{}, // TODO
					//Hostname: nil,
					//Subdomain: nil,
					HostAliases: nil, // TODO
					Affinity:    nil,
					Tolerations: nil,
					//SchedulerName: nil,
					//PriorityClassName: nil,
					Priority:                  nil,
					ReadinessGates:            nil, // TODO
					RuntimeClassName:          nil, // TODO
					EnableServiceLinks:        nil, // TODO should this better be false?
					PreemptionPolicy:          nil, // TODO
					Overhead:                  nil,
					TopologySpreadConstraints: nil,
					SetHostnameAsFQDN:         nil, // TODO
					OS:                        nil,
					HostUsers:                 nil,
					SchedulingGates:           nil,
					ResourceClaims:            []corev1.PodResourceClaim{},
				},
			},
		},
	}

	// Set the ownerRef for the Job
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/owners-dependents/
	if err := ctrl.SetControllerReference(pj, job, r.Scheme); err != nil {
		return nil, err
	}

	// create the cronjob
	log(
		"Creating a new Job",
		"Job.Namespace", job.Namespace,
		"Job.Name", job.Name,
	)
	if err := CreateOrUpdate(r, r, ctx, log, job, &batchv1.Job{}); err != nil {
		return nil, err
	}

	return job, nil
}

func addInitCommand(commands *string, command ...string) {
	if len(*commands) != 0 {
		*commands = *commands + " && "
	}
	*commands = *commands + strings.Join(command, " ")
}

func volumePresentAlready(new string, volumes []corev1.Volume) bool {
	for _, v := range volumes {
		if v.Name == new {
			return true
		}
	}
	return false
}

func getVolume(volume string, readOnly bool) corev1.Volume {
	return corev1.Volume{
		Name: volume,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: volume, // claim gets same name as volume it claims
				ReadOnly:  readOnly,
			},
		},
	}
}

func getVolumeMount(volume string, mountPath string) corev1.VolumeMount {
	return corev1.VolumeMount{
		Name:      volume,
		MountPath: mountPath,
	}
}

func getMountPath(stepId string) string {
	return "/vol/" + stepId
}

func isTrueInJob(j *batchv1.Job, conditionType batchv1.JobConditionType) bool {
	if j.Status.Conditions == nil {
		return false
	}
	for _, condition := range j.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}
