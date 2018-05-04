package e2e_test

import (
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	api "github.com/appscode/stash/apis/stash/v1alpha1"
	"github.com/appscode/stash/pkg/util"
	"github.com/appscode/stash/test/e2e/framework"
	. "github.com/appscode/stash/test/e2e/matcher"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1beta1"
	core "k8s.io/api/core/v1"
	extensions "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Snapshots", func() {
	var (
		err        error
		f          *framework.Invocation
		restic     api.Restic
		cred       core.Secret
		deployment apps.Deployment
		daemon     extensions.DaemonSet
		rc         core.ReplicationController
		rs         extensions.ReplicaSet
		ss         apps.StatefulSet
		svc        core.Service
		workload   api.LocalTypedReference
	)

	BeforeEach(func() {
		f = root.Invoke()
	})
	AfterEach(func() {
		f.DeleteDaemonSet(daemon.ObjectMeta)
		f.DeleteDeployment(deployment.ObjectMeta)
		f.DeleteReplicationController(rc.ObjectMeta)
		f.DeleteReplicaSet(rs.ObjectMeta)
		f.DeleteService(svc.ObjectMeta)
		f.DeleteStatefulSet(ss.ObjectMeta)
		f.DeleteRestic(restic.ObjectMeta)
		f.DeleteSecret(cred.ObjectMeta)
		f.DeleteRepositories(f.DaemonSetRepos(&daemon))
		f.DeleteRepositories(f.DeploymentRepos(&deployment))
		f.DeleteRepositories(f.ReplicationControllerRepos(&rc))
		f.DeleteRepositories(f.ReplicaSetRepos(&rs))
		f.DeleteRepositories(f.StatefulSetRepos(&ss))
		time.Sleep(60 * time.Second)
	})
	JustBeforeEach(func() {
		if missing, _ := BeZero().Match(cred); missing {
			Skip("Missing repository credential")
		}
		restic.Spec.Backend.StorageSecretName = cred.Name
		daemon = f.DaemonSet()
		deployment = f.Deployment()
		rc = f.ReplicationController()
		rs = f.ReplicaSet()
		ss = f.StatefulSet()
		svc = f.HeadlessService()

		// if a deployment's labels match to labels of replicaset, kubernetes make the deployment owner of the replicaset.
		// avoid this adding extra label to deployment.
		deployment.Labels["tier"] = "test"
		deployment.Spec.Template.Labels["tier"] = "test"
	})

	var (
		shouldCreateWorkloads = func() {
			By("Creating DaemonSet " + daemon.Name)
			_, err = f.CreateDaemonSet(daemon)
			Expect(err).NotTo(HaveOccurred())

			By("Creating Deployment " + deployment.Name)
			_, err = f.CreateDeployment(deployment)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicationController " + rc.Name)
			_, err = f.CreateReplicationController(rc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating ReplicaSet " + rs.Name)
			_, err = f.CreateReplicaSet(rs)
			Expect(err).NotTo(HaveOccurred())

			By("Creating service " + svc.Name)
			err = f.CreateService(svc)
			Expect(err).NotTo(HaveOccurred())

			By("Creating StatefulSet " + ss.Name)
			_, err = f.CreateStatefulSet(ss)
			Expect(err).NotTo(HaveOccurred())
		}

		shouldHaveSidecar = func() {
			By("Waiting for sidecar of DaemonSet")
			f.EventuallyDaemonSet(daemon.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for sidecar of Deployment")
			f.EventuallyDeployment(deployment.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for sidecar of ReplicationController")
			f.EventuallyReplicationController(rc.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for sidecar of ReplicaSet")
			f.EventuallyReplicaSet(rs.ObjectMeta).Should(HaveSidecar(util.StashContainer))

			By("Waiting for sidecar of StatefulSet")
			f.EventuallyStatefulSet(ss.ObjectMeta).Should(HaveSidecar(util.StashContainer))
		}

		shouldHaveRepositoryCRD = func() {
			By("Waiting for Repository CRD for DaemonSet")
			f.EventuallyRepository(&daemon).ShouldNot(BeEmpty())

			By("Waiting for Repository CRD for Deployment")
			f.EventuallyRepository(&deployment).ShouldNot(BeEmpty())

			By("Waiting for Repository CRD for ReplicationController")
			f.EventuallyRepository(&rc).ShouldNot(BeEmpty())

			By("Waiting for Repository CRD for ReplicaSet")
			f.EventuallyRepository(&rs).ShouldNot(BeEmpty())

			By("Waiting for Repository CRD for StatefulSet")
			f.EventuallyRepository(&ss).Should(WithTransform(func(repoList []*api.Repository) int {
				return len(repoList)
			}, BeNumerically("==", int(*ss.Spec.Replicas))))
		}

		shouldBackupComplete = func() {
			By("Waiting for backup to complete for DaemonsSet")
			f.EventuallyRepository(&daemon).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 2)))

			By("Waiting for backup to complete for Deployment")
			f.EventuallyRepository(&deployment).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 2)))

			By("Waiting for backup to complete for ReplicationController")
			f.EventuallyRepository(&rc).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 2)))

			By("Waiting for backup to complete for ReplicaSet")
			f.EventuallyRepository(&rs).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 2)))

			By("Waiting for backup to complete for StatefulSet")
			f.EventuallyRepository(&ss).Should(WithTransform(f.BackupCountInRepositoriesStatus, BeNumerically(">=", 2)))
		}

		performOperationOnSnapshot = func() {
			By("Creating repository Secret " + cred.Name)
			err = f.CreateSecret(cred)
			Expect(err).NotTo(HaveOccurred())

			By("Creating restic")
			err = f.CreateRestic(restic)
			Expect(err).NotTo(HaveOccurred())

			By("Creating workloads")
			shouldCreateWorkloads()

			By("Waiting for workloads to have sidecar")
			shouldHaveSidecar()

			By("Waiting for Repository CRD")
			shouldHaveRepositoryCRD()

			By("Waiting for backup to complete")
			shouldBackupComplete()

			By("Listing all snapshots")
			_, err := f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{})
			Expect(err).NotTo(HaveOccurred())

			By("Get a particular snapshot")
			snapshots, err := f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "workload-kind=Deployment"})
			singleSnapshot, err := f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).Get(snapshots.Items[len(snapshots.Items)-1].Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(singleSnapshot.Name).To(BeEquivalentTo(snapshots.Items[len(snapshots.Items)-1].Name))

			By("Filter by workload kind")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "workload-kind=Deployment"})
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshots).Should(HavePrefixInName("deployment"))

			By("Filter by workload name")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "workload-name=" + deployment.Name})
			Expect(err).NotTo(HaveOccurred())
			workload.Kind = api.KindDeployment
			workload.Name = deployment.Name
			Expect(snapshots).Should(HavePrefixInName(workload.GetRepositoryCRDName("", "")))

			By("Filter by pod name")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "pod-name=" + ss.Name + "-0"})
			Expect(err).NotTo(HaveOccurred())
			workload.Kind = api.KindStatefulSet
			workload.Name = ss.Name
			Expect(snapshots).Should(HavePrefixInName(workload.GetRepositoryCRDName(ss.Name+"-0", "")))

			nodename := os.Getenv("NODE_NAME")
			if nodename == "" {
				nodename = "minikube"
			}
			By("Filter by node name")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "node-name=" + nodename})
			Expect(err).NotTo(HaveOccurred())
			workload.Kind = api.KindDaemonSet
			workload.Name = daemon.Name
			Expect(snapshots).Should(HavePrefixInName(workload.GetRepositoryCRDName("", nodename)))

			workload.Kind = api.KindDeployment
			workload.Name = deployment.Name
			reponame := workload.GetRepositoryCRDName("", "")

			By("Filter by repository name")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "repository=" + reponame})
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshots).Should(HavePrefixInName(reponame))

			By("Filter by negated selector")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "repository!=" + reponame})
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshots).ShouldNot(HavePrefixInName(reponame))

			By("Filter by set based selector")
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "repository in(" + reponame + ")"})
			Expect(err).NotTo(HaveOccurred())
			Expect(snapshots).Should(HavePrefixInName(reponame))

			By("Deleting snapshot " + snapshots.Items[len(snapshots.Items)-1].Name)
			snapshots, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).List(metav1.ListOptions{LabelSelector: "workload-kind=Deployment"})
			Expect(err).NotTo(HaveOccurred())
			snapshotToDelete := snapshots.Items[len(snapshots.Items)-1].Name
			policy := metav1.DeletePropagationForeground
			err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).Delete(snapshotToDelete, &metav1.DeleteOptions{PropagationPolicy: &policy})
			Expect(err).NotTo(HaveOccurred())

			By("Checking deleted snapshot not exist")
			singleSnapshot, err = f.StashClient.RepositoriesV1alpha1().Snapshots(f.Namespace()).Get(snapshotToDelete, metav1.GetOptions{})
			Expect(err).To(HaveOccurred())
		}
	)

	Describe("Snapshots operations", func() {
		Context(`"Local" backend`, func() {
			AfterEach(func() {
				framework.CleanupMinikubeHostPath()
			})
			BeforeEach(func() {
				cred = f.SecretForLocalBackend()
				restic = f.ResticForLocalBackend()
			})
			It(`should success to perform Snapshot's operations`, performOperationOnSnapshot)

		})
		Context(`"Minio" backend`, func() {
			AfterEach(func() {
				f.DeleteMinioServer()
			})
			BeforeEach(func() {
				By("Checking if Restic installed in /bin directory")
				cmd := exec.Command("restic")
				err := cmd.Run()
				if err != nil {
					Skip("restic executable not found in /bin directory. Please install in /bin directory from: https://github.com/restic/restic/releases")
				}

				minikubeIP := net.IP{192, 168, 99, 100}

				By("Creating Minio server without cacert")
				_, err = f.CreateMinioServer(true, []net.IP{minikubeIP})
				Expect(err).NotTo(HaveOccurred())

				msvc, err := f.KubeClient.CoreV1().Services(f.Namespace()).Get("minio-service", metav1.GetOptions{})
				Expect(err).NotTo(HaveOccurred())
				minioServiceNodePort := strconv.Itoa(int(msvc.Spec.Ports[0].NodePort))

				restic = f.ResticForMinioBackend("https://" + minikubeIP.String() + ":" + minioServiceNodePort)
				cred = f.SecretForMinioBackend(true)
			})
			It(`should success to perform Snapshot's operations`, performOperationOnSnapshot)

		})
	})

})
