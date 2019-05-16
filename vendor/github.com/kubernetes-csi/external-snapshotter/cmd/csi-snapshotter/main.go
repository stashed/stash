/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"time"

	"google.golang.org/grpc"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/kubernetes-csi/csi-lib-utils/connection"
	"github.com/kubernetes-csi/csi-lib-utils/leaderelection"
	csirpc "github.com/kubernetes-csi/csi-lib-utils/rpc"
	"github.com/kubernetes-csi/external-snapshotter/pkg/controller"
	"github.com/kubernetes-csi/external-snapshotter/pkg/snapshotter"

	clientset "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned"
	snapshotscheme "github.com/kubernetes-csi/external-snapshotter/pkg/client/clientset/versioned/scheme"
	informers "github.com/kubernetes-csi/external-snapshotter/pkg/client/informers/externalversions"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	coreinformers "k8s.io/client-go/informers"
)

const (
	// Number of worker threads
	threads = 10

	// Default timeout of short CSI calls like GetPluginInfo
	defaultCSITimeout = time.Minute
)

// Command line flags
var (
	snapshotterName                 = flag.String("snapshotter", "", "This option is deprecated.")
	kubeconfig                      = flag.String("kubeconfig", "", "Absolute path to the kubeconfig file. Required only when running out of cluster.")
	connectionTimeout               = flag.Duration("connection-timeout", 0, "The --connection-timeout flag is deprecated")
	csiAddress                      = flag.String("csi-address", "/run/csi/socket", "Address of the CSI driver socket.")
	createSnapshotContentRetryCount = flag.Int("create-snapshotcontent-retrycount", 5, "Number of retries when we create a snapshot content object for a snapshot.")
	createSnapshotContentInterval   = flag.Duration("create-snapshotcontent-interval", 10*time.Second, "Interval between retries when we create a snapshot content object for a snapshot.")
	resyncPeriod                    = flag.Duration("resync-period", 60*time.Second, "Resync interval of the controller.")
	snapshotNamePrefix              = flag.String("snapshot-name-prefix", "snapshot", "Prefix to apply to the name of a created snapshot")
	snapshotNameUUIDLength          = flag.Int("snapshot-name-uuid-length", -1, "Length in characters for the generated uuid of a created snapshot. Defaults behavior is to NOT truncate.")
	showVersion                     = flag.Bool("version", false, "Show version.")
	csiTimeout                      = flag.Duration("timeout", defaultCSITimeout, "The timeout for any RPCs to the CSI driver. Default is 10s.")

	leaderElection          = flag.Bool("leader-election", false, "Enables leader election.")
	leaderElectionNamespace = flag.String("leader-election-namespace", "", "The namespace where the leader election resource exists. Defaults to the pod namespace if not set.")
)

var (
	version                = "unknown"
	leaderElectionLockName = "external-snapshotter-leader-election"
)

func main() {
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()

	if *showVersion {
		fmt.Println(os.Args[0], version)
		os.Exit(0)
	}
	klog.Infof("Version: %s", version)

	if *connectionTimeout != 0 {
		klog.Warning("--connection-timeout is deprecated and will have no effect")
	}

	if *snapshotterName != "" {
		klog.Warning("--snapshotter is deprecated and will have no effect")
	}

	// Create the client config. Use kubeconfig if given, otherwise assume in-cluster.
	config, err := buildConfig(*kubeconfig)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	snapClient, err := clientset.NewForConfig(config)
	if err != nil {
		klog.Errorf("Error building snapshot clientset: %s", err.Error())
		os.Exit(1)
	}

	factory := informers.NewSharedInformerFactory(snapClient, *resyncPeriod)
	coreFactory := coreinformers.NewSharedInformerFactory(kubeClient, *resyncPeriod)

	// Create CRD resource
	aeclientset, err := apiextensionsclient.NewForConfig(config)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// initialize CRD resource if it does not exist
	err = CreateCRD(aeclientset)
	if err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}

	// Add Snapshot types to the defualt Kubernetes so events can be logged for them
	snapshotscheme.AddToScheme(scheme.Scheme)

	// Connect to CSI.
	csiConn, err := connection.Connect(*csiAddress)
	if err != nil {
		klog.Errorf("error connecting to CSI driver: %v", err)
		os.Exit(1)
	}

	// Pass a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), *csiTimeout)
	defer cancel()

	// Find driver name
	*snapshotterName, err = csirpc.GetDriverName(ctx, csiConn)
	if err != nil {
		klog.Errorf("error getting CSI driver name: %v", err)
		os.Exit(1)
	}

	klog.V(2).Infof("CSI driver name: %q", *snapshotterName)

	// Check it's ready
	if err = csirpc.ProbeForever(csiConn, *csiTimeout); err != nil {
		klog.Errorf("error waiting for CSI driver to be ready: %v", err)
		os.Exit(1)
	}

	// Find out if the driver supports create/delete snapshot.
	supportsCreateSnapshot, err := supportsControllerCreateSnapshot(ctx, csiConn)
	if err != nil {
		klog.Errorf("error determining if driver supports create/delete snapshot operations: %v", err)
		os.Exit(1)
	}
	if !supportsCreateSnapshot {
		klog.Errorf("CSI driver %s does not support ControllerCreateSnapshot", *snapshotterName)
		os.Exit(1)
	}

	if len(*snapshotNamePrefix) == 0 {
		klog.Error("Snapshot name prefix cannot be of length 0")
		os.Exit(1)
	}

	klog.V(2).Infof("Start NewCSISnapshotController with snapshotter [%s] kubeconfig [%s] connectionTimeout [%+v] csiAddress [%s] createSnapshotContentRetryCount [%d] createSnapshotContentInterval [%+v] resyncPeriod [%+v] snapshotNamePrefix [%s] snapshotNameUUIDLength [%d]", *snapshotterName, *kubeconfig, *connectionTimeout, *csiAddress, createSnapshotContentRetryCount, *createSnapshotContentInterval, *resyncPeriod, *snapshotNamePrefix, snapshotNameUUIDLength)

	snapShotter := snapshotter.NewSnapshotter(csiConn)
	ctrl := controller.NewCSISnapshotController(
		snapClient,
		kubeClient,
		*snapshotterName,
		factory.Volumesnapshot().V1alpha1().VolumeSnapshots(),
		factory.Volumesnapshot().V1alpha1().VolumeSnapshotContents(),
		factory.Volumesnapshot().V1alpha1().VolumeSnapshotClasses(),
		coreFactory.Core().V1().PersistentVolumeClaims(),
		*createSnapshotContentRetryCount,
		*createSnapshotContentInterval,
		snapShotter,
		*csiTimeout,
		*resyncPeriod,
		*snapshotNamePrefix,
		*snapshotNameUUIDLength,
	)

	run := func(context.Context) {
		// run...
		stopCh := make(chan struct{})
		factory.Start(stopCh)
		coreFactory.Start(stopCh)
		go ctrl.Run(threads, stopCh)

		// ...until SIGINT
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(stopCh)
	}

	if !*leaderElection {
		run(context.TODO())
	} else {
		le := leaderelection.NewLeaderElection(kubeClient, leaderElectionLockName, run)
		if *leaderElectionNamespace != "" {
			le.WithNamespace(*leaderElectionNamespace)
		}
		if err := le.Run(); err != nil {
			klog.Fatalf("failed to initialize leader election: %v", err)
		}
	}
}

func buildConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

func supportsControllerCreateSnapshot(ctx context.Context, conn *grpc.ClientConn) (bool, error) {
	capabilities, err := csirpc.GetControllerCapabilities(ctx, conn)
	if err != nil {
		return false, err
	}

	return capabilities[csi.ControllerServiceCapability_RPC_CREATE_DELETE_SNAPSHOT], nil
}
