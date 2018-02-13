package e2e

import (
	// Hack!!! Tell vc to not cleanup the vendored codes
	_ "github.com/appscode/go/log/golog"
	_ "github.com/appscode/kutil/apps/v1beta1"
	_ "github.com/appscode/kutil/core/v1"
	_ "github.com/appscode/kutil/extensions/v1beta1"
	_ "github.com/appscode/kutil/tools/certstore"
	_ "github.com/appscode/kutil/tools/clientcmd"
)
