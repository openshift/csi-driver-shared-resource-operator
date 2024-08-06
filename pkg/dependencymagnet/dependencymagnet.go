//go:build tools
// +build tools

package dependencymagnet

import (
	_ "github.com/openshift/api/sharedresource/v1alpha1/zz_generated.crd-manifests"
	_ "github.com/openshift/build-machinery-go"
)
