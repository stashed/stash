package framework

import (
	"database/sql"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"k8s.io/apimachinery/pkg/api/resource"

	_ "github.com/go-sql-driver/mysql"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	apps "k8s.io/api/apps/v1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apps_util "kmodules.xyz/client-go/apps/v1"
	appCatalog "kmodules.xyz/custom-resources/apis/appcatalog/v1alpha1"
)

const (
	KeyUser     = "user"
	KeyPassword = "password"
	SuperUser   = "root"

	KeyMySQLRootPassword   = "MYSQL_ROOT_PASSWORD"
	MySQLServingPortName   = "mysql"
	MySQLServingPortNumber = 33060
)

func (f *Invocation) MySQLCredentials() *core.Secret {
	return &core.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Data: map[string][]byte{
			KeyUser:     []byte(SuperUser),
			KeyPassword: []byte(f.app),
		},
		Type: core.SecretTypeOpaque,
	}
}

func (f *Invocation) MySQLService() *core.Service {
	return &core.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: core.ServiceSpec{
			Selector: map[string]string{
				"app": f.app,
			},
			Ports: []core.ServicePort{
				{
					Name: MySQLServingPortName,
					Port: MySQLServingPortNumber,
				},
			},
		},
	}
}

func (f *Invocation) MySQLPVC() *core.PersistentVolumeClaim {
	return &core.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: core.PersistentVolumeClaimSpec{
			AccessModes: []core.PersistentVolumeAccessMode{
				core.ReadWriteOnce,
			},
			Resources: core.ResourceRequirements{
				Requests: core.ResourceList{
					core.ResourceStorage: resource.MustParse("128Mi"),
				},
			},
		},
	}
}

func (f *Invocation) MySQLDeployment(cred *core.Secret, pvc *core.PersistentVolumeClaim) *apps.Deployment {
	label := map[string]string{
		"app": f.app,
	}
	return &apps.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: apps.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: label,
			},
			Template: core.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: label,
				},
				Spec: core.PodSpec{
					Containers: []core.Container{
						{
							Name:  "mysql",
							Image: "mysql:8.0.14",
							Env: []core.EnvVar{
								{
									Name: KeyMySQLRootPassword,
									ValueFrom: &core.EnvVarSource{
										SecretKeyRef: &core.SecretKeySelector{
											LocalObjectReference: core.LocalObjectReference{
												Name: cred.Name,
											},
											Key: KeyPassword,
										},
									},
								},
							},
							Ports: []core.ContainerPort{
								{
									Name:          MySQLServingPortName,
									ContainerPort: MySQLServingPortNumber,
								},
							},
							VolumeMounts: []core.VolumeMount{
								{
									Name:      pvc.Name,
									MountPath: "/var/lib/mysql",
								},
							},
						},
					},
					Volumes: []core.Volume{
						{
							Name: pvc.Name,
							VolumeSource: core.VolumeSource{
								PersistentVolumeClaim: &core.PersistentVolumeClaimVolumeSource{
									ClaimName: pvc.Name,
								},
							},
						},
					},
				},
			},
		},
	}
}

func (f *Invocation) MySQLAppBinding(cred *core.Secret, svc *core.Service) *appCatalog.AppBinding {
	return &appCatalog.AppBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      f.app,
			Namespace: f.namespace,
		},
		Spec: appCatalog.AppBindingSpec{
			Type:    "mysql",
			Version: "8.0.14",
			ClientConfig: appCatalog.ClientConfig{
				Service: &appCatalog.ServiceReference{
					Scheme: "mysql",
					Name:   svc.Name,
					Port:   MySQLServingPortNumber,
				},
			},
			Secret: &core.LocalObjectReference{
				Name: cred.Name,
			},
		},
	}
}

func (f *Invocation) DeployMySQLDatabase() (*apps.Deployment, error) {
	By("Creating Secret for MySQL")
	cred := f.MySQLCredentials()
	_, err := f.CreateSecret(*cred)
	Expect(err).NotTo(HaveOccurred())

	By("Creating PVC for MySQL")
	pvc := f.MySQLPVC()
	_, err = f.CreatePersistentVolumeClaim(pvc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating Service for MySQL")
	svc := f.MySQLService()
	_, err = f.CreateService(*svc)
	Expect(err).NotTo(HaveOccurred())

	By("Creating MySQL")
	dpl := f.MySQLDeployment(cred, pvc)
	dpl, err = f.CreateDeployment(*dpl)
	Expect(err).NotTo(HaveOccurred())

	By("Waiting for MySQL Deployment to be ready")
	err = apps_util.WaitUntilDeploymentReady(f.KubeClient, dpl.ObjectMeta)
	Expect(err).NotTo(HaveOccurred())

	By("Creating AppBinding for the MySQL")
	appBinding := f.MySQLAppBinding(cred, svc)
	appBinding, err = f.createAppBinding(appBinding)
	Expect(err).NotTo(HaveOccurred())

	f.AppendToCleanupList(appBinding, dpl, svc, pvc, cred)
	return dpl, nil
}

func (f *Invocation) EventuallyConnectWithMySQLServer(db *sql.DB) error {
	return wait.PollImmediate(2*time.Second, 3*time.Minute, func() (bool, error) {
		if err := db.Ping(); err != nil {
			fmt.Println(err)
			return false, nil // don't return error. we need to retry.
		}
		return true, nil
	})
}

func (f *Invocation) createAppBinding(appBinding *appCatalog.AppBinding) (*appCatalog.AppBinding, error) {
	return f.catalogClient.AppcatalogV1alpha1().AppBindings(appBinding.Namespace).Create(appBinding)
}

func (f *Invocation) CreateMySQLTable(db *sql.DB, tableName string) error {
	stmnt, err := db.Prepare(fmt.Sprintf("CREATE TABLE %s ( ID int );"))
	if err != nil {
		return err
	}
	defer stmnt.Close()

	result, err := stmnt.Exec()
	if err != nil {
		return err
	}
	fmt.Println("========= Result: ", result)
	return nil
}
