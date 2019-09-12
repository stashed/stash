package framework

import (
	"github.com/appscode/go/crypto/rand"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	store "kmodules.xyz/objectstore-api/api/v1"
	"stash.appscode.dev/stash/apis/stash/v1alpha1"
	"stash.appscode.dev/stash/apis/stash/v1beta1"
)

func (f *Invocation) GetBackupBlueprint(secret string) v1beta1.BackupBlueprint {

	return v1beta1.BackupBlueprint{
		ObjectMeta: metav1.ObjectMeta{
			Name: rand.WithUniqSuffix(f.app),
		},
		Spec: v1beta1.BackupBlueprintSpec{
			RepositorySpec: v1alpha1.RepositorySpec{
				Backend: store.Backend{
					GCS: &store.GCSSpec{
						Bucket: "appscode-qa",
						Prefix: "stash-backup/${TARGET_NAMESPACE}/${TARGET_KIND}/${TARGET_NAME}",
					},
					StorageSecretName: secret,
				},
			},
			Schedule: "*/2 * * * *",
			RetentionPolicy: v1alpha1.RetentionPolicy{
				Name:     "keep-last-5",
				KeepLast: 5,
				Prune:    true,
			},
		},
	}
}

func (f *Framework) CreateBackupBlueprint(backupBlueprint v1beta1.BackupBlueprint) (*v1beta1.BackupBlueprint, error) {
	return f.StashClient.StashV1beta1().BackupBlueprints().Create(&backupBlueprint)
}

func (f *Invocation) DeleteBackupBlueprint(name string) error {
	return f.StashClient.StashV1beta1().BackupBlueprints().Delete(name, &metav1.DeleteOptions{})
}
