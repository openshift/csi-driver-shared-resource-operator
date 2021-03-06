// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	internalinterfaces "github.com/openshift/client-go/sharedresource/informers/externalversions/internalinterfaces"
)

// Interface provides access to all the informers in this group version.
type Interface interface {
	// SharedConfigMaps returns a SharedConfigMapInformer.
	SharedConfigMaps() SharedConfigMapInformer
	// SharedSecrets returns a SharedSecretInformer.
	SharedSecrets() SharedSecretInformer
}

type version struct {
	factory          internalinterfaces.SharedInformerFactory
	namespace        string
	tweakListOptions internalinterfaces.TweakListOptionsFunc
}

// New returns a new Interface.
func New(f internalinterfaces.SharedInformerFactory, namespace string, tweakListOptions internalinterfaces.TweakListOptionsFunc) Interface {
	return &version{factory: f, namespace: namespace, tweakListOptions: tweakListOptions}
}

// SharedConfigMaps returns a SharedConfigMapInformer.
func (v *version) SharedConfigMaps() SharedConfigMapInformer {
	return &sharedConfigMapInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}

// SharedSecrets returns a SharedSecretInformer.
func (v *version) SharedSecrets() SharedSecretInformer {
	return &sharedSecretInformer{factory: v.factory, tweakListOptions: v.tweakListOptions}
}
