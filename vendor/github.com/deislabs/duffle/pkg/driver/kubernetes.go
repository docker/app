package driver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/deislabs/cnab-go/driver"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	batchclientv1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	coreclientv1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	k8sContainerName    = "invocation"
	k8sFileSecretVolume = "files"
)

// KubernetesDriver runs an invocation image in a Kubernetes cluster.
type KubernetesDriver struct {
	Namespace             string
	ServiceAccountName    string
	LimitCPU              resource.Quantity
	LimitMemory           resource.Quantity
	ActiveDeadlineSeconds int64
	BackoffLimit          int32
	SkipCleanup           bool
	skipJobStatusCheck    bool
	jobs                  batchclientv1.JobInterface
	secrets               coreclientv1.SecretInterface
	pods                  coreclientv1.PodInterface
	deletionPolicy        metav1.DeletionPropagation
	requiredCompletions   int32
}

// NewKubernetesDriver initializes a Kubernetes driver.
func NewKubernetesDriver(namespace, serviceAccount string, conf *rest.Config) (*KubernetesDriver, error) {
	driver := &KubernetesDriver{
		Namespace:          namespace,
		ServiceAccountName: serviceAccount,
	}
	driver.setDefaults()
	err := driver.setClient(conf)
	return driver, err
}

// Handles receives an ImageType* and answers whether this driver supports that type.
func (k *KubernetesDriver) Handles(imagetype string) bool {
	return imagetype == driver.ImageTypeDocker || imagetype == driver.ImageTypeOCI
}

// Config returns the Kubernetes driver configuration options.
func (k *KubernetesDriver) Config() map[string]string {
	return map[string]string{
		"KUBE_NAMESPACE":  "Kubernetes namespace in which to run the invocation image",
		"SERVICE_ACCOUNT": "Kubernetes service account to be mounted by the invocation image (if empty, no service account token will be mounted)",
		"KUBE_CONFIG":     "Absolute path to the kubeconfig file",
		"MASTER_URL":      "Kubernetes master endpoint",
	}
}

// SetConfig sets Kubernetes driver configuration.
func (k *KubernetesDriver) SetConfig(settings map[string]string) {
	k.setDefaults()
	k.Namespace = settings["KUBE_NAMESPACE"]
	k.ServiceAccountName = settings["SERVICE_ACCOUNT"]

	var kubeconfig string
	if kpath := settings["KUBE_CONFIG"]; kpath != "" {
		kubeconfig = kpath
	} else if home := homeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	conf, err := clientcmd.BuildConfigFromFlags(settings["MASTER_URL"], kubeconfig)
	if err != nil {
		panic(err)
	}
	err = k.setClient(conf)
	if err != nil {
		panic(err)
	}
}

func (k *KubernetesDriver) setDefaults() {
	k.SkipCleanup = false
	k.BackoffLimit = 0
	k.ActiveDeadlineSeconds = 300
	k.requiredCompletions = 1
	k.deletionPolicy = metav1.DeletePropagationBackground
}

func (k *KubernetesDriver) setClient(conf *rest.Config) error {
	coreClient, err := coreclientv1.NewForConfig(conf)
	if err != nil {
		return err
	}
	batchClient, err := batchclientv1.NewForConfig(conf)
	if err != nil {
		return err
	}
	k.jobs = batchClient.Jobs(k.Namespace)
	k.secrets = coreClient.Secrets(k.Namespace)
	k.pods = coreClient.Pods(k.Namespace)

	return nil
}

// Run executes the operation inside of the invocation image.
func (k *KubernetesDriver) Run(op *driver.Operation) error {
	if k.Namespace == "" {
		return fmt.Errorf("KUBE_NAMESPACE is required")
	}
	labelMap := generateLabels(op)
	meta := metav1.ObjectMeta{
		Namespace:    k.Namespace,
		GenerateName: generateNameTemplate(op),
		Labels:       labelMap,
	}
	// Mount SA token if a non-zero value for ServiceAccountName has been specified
	mountServiceAccountToken := k.ServiceAccountName != ""
	job := &batchv1.Job{
		ObjectMeta: meta,
		Spec: batchv1.JobSpec{
			ActiveDeadlineSeconds: &k.ActiveDeadlineSeconds,
			Completions:           &k.requiredCompletions,
			BackoffLimit:          &k.BackoffLimit,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labelMap,
				},
				Spec: v1.PodSpec{
					ServiceAccountName:           k.ServiceAccountName,
					AutomountServiceAccountToken: &mountServiceAccountToken,
					RestartPolicy:                v1.RestartPolicyNever,
				},
			},
		},
	}
	container := v1.Container{
		Name:    k8sContainerName,
		Image:   op.Image,
		Command: []string{"/cnab/app/run"},
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				v1.ResourceCPU:    k.LimitCPU,
				v1.ResourceMemory: k.LimitMemory,
			},
		},
		ImagePullPolicy: v1.PullIfNotPresent,
	}

	if len(op.Environment) > 0 {
		secret := &v1.Secret{
			ObjectMeta: meta,
			StringData: op.Environment,
		}
		secret.ObjectMeta.GenerateName += "env-"
		envsecret, err := k.secrets.Create(secret)
		if err != nil {
			return err
		}
		if !k.SkipCleanup {
			defer k.deleteSecret(envsecret.ObjectMeta.Name)
		}

		container.EnvFrom = []v1.EnvFromSource{
			{
				SecretRef: &v1.SecretEnvSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: envsecret.ObjectMeta.Name,
					},
				},
			},
		}
	}

	if len(op.Files) > 0 {
		secret, mounts := generateFileSecret(op.Files)
		secret.ObjectMeta = metav1.ObjectMeta{
			Namespace:    k.Namespace,
			GenerateName: generateNameTemplate(op) + "files-",
			Labels:       labelMap,
		}
		secret, err := k.secrets.Create(secret)
		if err != nil {
			return err
		}
		if !k.SkipCleanup {
			defer k.deleteSecret(secret.ObjectMeta.Name)
		}

		job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, v1.Volume{
			Name: k8sFileSecretVolume,
			VolumeSource: v1.VolumeSource{
				Secret: &v1.SecretVolumeSource{
					SecretName: secret.ObjectMeta.Name,
				},
			},
		})
		container.VolumeMounts = mounts
	}

	job.Spec.Template.Spec.Containers = []v1.Container{container}
	job, err := k.jobs.Create(job)
	if err != nil {
		return err
	}
	if !k.SkipCleanup {
		defer k.deleteJob(job.ObjectMeta.Name)
	}

	// Return early for unit testing purposes (the fake k8s client implementation just
	// hangs during watch because no events are ever created on the Job)
	if k.skipJobStatusCheck {
		return nil
	}

	selector := metav1.ListOptions{
		LabelSelector: labels.Set(job.ObjectMeta.Labels).String(),
	}

	return k.watchJobStatusAndLogs(selector, op.Out)
}

func (k *KubernetesDriver) watchJobStatusAndLogs(selector metav1.ListOptions, out io.Writer) error {
	// Stream Pod logs in the background
	logsStreamingComplete := make(chan bool)
	err := k.streamPodLogs(selector, out, logsStreamingComplete)
	if err != nil {
		return err
	}
	// Watch job events and exit on failure/success
	watch, err := k.jobs.Watch(selector)
	if err != nil {
		return err
	}
	for event := range watch.ResultChan() {
		job, ok := event.Object.(*batchv1.Job)
		if !ok {
			return fmt.Errorf("unexpected type")
		}
		complete := false
		for _, cond := range job.Status.Conditions {
			if cond.Type == batchv1.JobFailed {
				err = fmt.Errorf(cond.Message)
				complete = true
				break
			}
			if cond.Type == batchv1.JobComplete {
				complete = true
				break
			}
		}
		if complete {
			break
		}
	}
	if err != nil {
		return err
	}

	// Wait for pod logs to finish printing
	for i := 0; i < int(k.requiredCompletions); i++ {
		<-logsStreamingComplete
	}

	return nil
}

func (k *KubernetesDriver) streamPodLogs(options metav1.ListOptions, out io.Writer, done chan bool) error {
	watcher, err := k.pods.Watch(options)
	if err != nil {
		return err
	}

	go func() {
		// Track pods whose logs have been streamed by pod name. We need to know when we've already
		// processed logs for a given pod, since multiple lifecycle events are received per pod.
		streamedLogs := map[string]bool{}
		for event := range watcher.ResultChan() {
			pod, ok := event.Object.(*v1.Pod)
			if !ok {
				continue
			}
			podName := pod.GetName()
			if streamedLogs[podName] {
				// The event was for a pod whose logs have already been streamed, so do nothing.
				continue
			}
			req := k.pods.GetLogs(podName, &v1.PodLogOptions{
				Container: k8sContainerName,
				Follow:    true,
			})
			reader, err := req.Stream()
			// There was an error connecting to the pod, so continue the loop and attempt streaming
			// logs again next time there is an event for the same pod.
			if err != nil {
				continue
			}

			// We successfully connected to the pod, so mark it as having streamed logs.
			streamedLogs[podName] = true
			// Block the loop until all logs from the pod have been processed.
			io.Copy(out, reader)
			reader.Close()
			done <- true
		}
	}()

	return nil
}

func (k *KubernetesDriver) deleteSecret(name string) error {
	return k.secrets.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &k.deletionPolicy,
	})
}

func (k *KubernetesDriver) deleteJob(name string) error {
	return k.jobs.Delete(name, &metav1.DeleteOptions{
		PropagationPolicy: &k.deletionPolicy,
	})
}

func generateNameTemplate(op *driver.Operation) string {
	return fmt.Sprintf("%s-%s-", op.Installation, op.Action)
}

func generateLabels(op *driver.Operation) map[string]string {
	return map[string]string{
		"cnab.io/installation": op.Installation,
		"cnab.io/action":       op.Action,
		"cnab.io/revision":     op.Revision,
	}
}

func generateFileSecret(files map[string]string) (*v1.Secret, []v1.VolumeMount) {
	size := len(files)
	data := make(map[string]string, size)
	mounts := make([]v1.VolumeMount, size)

	i := 0
	for path, contents := range files {
		key := strings.Replace(filepath.ToSlash(path), "/", "_", -1)
		data[key] = contents
		mounts[i] = v1.VolumeMount{
			Name:      k8sFileSecretVolume,
			MountPath: path,
			SubPath:   key,
		}
		i++
	}

	secret := &v1.Secret{
		StringData: data,
	}

	return secret, mounts
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE") // windows
}
