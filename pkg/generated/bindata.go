// Code generated for package generated by go-bindata DO NOT EDIT. (@generated)
// sources:
// assets/csidriver.yaml
// assets/metrics_service.yaml
// assets/node.yaml
// assets/node_sa.yaml
// assets/rbac/node_binding.yaml
// assets/rbac/node_privileged_binding.yaml
// assets/rbac/node_role.yaml
// assets/rbac/privileged_role.yaml
// assets/rbac/prometheus_role.yaml
// assets/rbac/prometheus_rolebinding.yaml
// assets/service.yaml
// assets/servicemonitor.yaml
package generated

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type asset struct {
	bytes []byte
	info  os.FileInfo
}

type bindataFileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

// Name return file name
func (fi bindataFileInfo) Name() string {
	return fi.name
}

// Size return file size
func (fi bindataFileInfo) Size() int64 {
	return fi.size
}

// Mode return file mode
func (fi bindataFileInfo) Mode() os.FileMode {
	return fi.mode
}

// Mode return file modify time
func (fi bindataFileInfo) ModTime() time.Time {
	return fi.modTime
}

// IsDir return file whether a directory
func (fi bindataFileInfo) IsDir() bool {
	return fi.mode&os.ModeDir != 0
}

// Sys return file is sys mode
func (fi bindataFileInfo) Sys() interface{} {
	return nil
}

var _csidriverYaml = []byte(`apiVersion: storage.k8s.io/v1
kind: CSIDriver
metadata:
  name: csi.sharedresource.openshift.io
  annotations:
    # This CSIDriver is managed by an OCP CSI operator
    csi.openshift.io/managed: "true"
spec:
  # Supports ephemeral inline volumes.
  volumeLifecycleModes:
    - Ephemeral
  # To determine at runtime which mode a volume uses, pod info and its
  # "csi.storage.k8s.io/ephemeral" entry are needed.
  podInfoOnMount: true
  # Always apply pod.spec.securityContext.fsGroup, autodetection does not work for Ephemeral volumes.
  fsGroupPolicy: File
  # This CSI driver does not implement ControllerPublish.
  attachRequired: false
`)

func csidriverYamlBytes() ([]byte, error) {
	return _csidriverYaml, nil
}

func csidriverYaml() (*asset, error) {
	bytes, err := csidriverYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "csidriver.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _metrics_serviceYaml = []byte(`kind: Service
apiVersion: v1
metadata:
  name: shared-resource-csi-driver-node-metrics
  namespace: openshift-cluster-csi-drivers
  labels:
    app: shared-resource-csi-driver-node
spec:
  selector:
    app: shared-resource-csi-driver-node
  ports:
    - name: metrics
      port: 6000
`)

func metrics_serviceYamlBytes() ([]byte, error) {
	return _metrics_serviceYaml, nil
}

func metrics_serviceYaml() (*asset, error) {
	bytes, err := metrics_serviceYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "metrics_service.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _nodeYaml = []byte(`kind: DaemonSet
apiVersion: apps/v1
metadata:
  name: shared-resource-csi-driver-node
  namespace: openshift-cluster-csi-drivers
  labels:
    app: shared-resource-csi-driver-node
spec:
  selector:
    matchLabels:
      app: shared-resource-csi-driver-node
  template:
    metadata:
      labels:
        app: shared-resource-csi-driver-node
    spec:
      serviceAccountName: csi-driver-shared-resource-plugin
      containers:
        - name: node-driver-registrar
          image: ${NODE_DRIVER_REGISTRAR_IMAGE}
          args:
            - --v=5
            - --csi-address=/csi/csi.sock
            - --kubelet-registration-path=/var/lib/kubelet/plugins/csi-hostpath/csi.sock
          securityContext:
            # This is necessary only for systems with SELinux, where
            # non-privileged sidecar containers cannot access unix domain socket
            # created by privileged CSI driver container.
            privileged: true
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
          volumeMounts:
            - mountPath: /csi
              name: socket-dir
            - mountPath: /registration
              name: registration-dir
            - mountPath: /csi-data-dir
              name: csi-data-dir

        - name: hostpath
          image: ${DRIVER_IMAGE}
          # for development purposes; eventually switch to IfNotPresent
          imagePullPolicy: Always
          command:
            - csi-driver-shared-resource
          args:
            - --config=/var/run/configmaps/config/config.yaml
            - "--drivername=csi.sharedresource.openshift.io"
            - "--v=4"
            - "--nodeid=$(KUBE_NODE_NAME)"
          env:
            - name: KUBE_NODE_NAME
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: spec.nodeName
          securityContext:
            privileged: true
          ports:
            - containerPort: 9898
              name: healthz
              protocol: TCP
            - containerPort: 6000
              name: metrics
              protocol: TCP
          volumeMounts:
            - mountPath: /var/run/configmaps/config
              name: config
            - mountPath: /csi
              name: socket-dir
            - mountPath: /var/lib/kubelet/pods
              mountPropagation: Bidirectional
              name: mountpoint-dir
            - mountPath: /var/lib/kubelet/plugins
              mountPropagation: Bidirectional
              name: plugins-dir
            - mountPath: /csi-data-dir
              name: csi-data-dir
            - mountPath: /csi-volumes-map
              name: csi-volumes-map
            - mountPath: /dev
              name: dev-dir

      volumes:
        - configMap:
            optional: true
            name: csi-driver-shared-resource-config
          name: config
        - hostPath:
            path: /var/lib/kubelet/plugins/csi-hostpath
            type: DirectoryOrCreate
          name: socket-dir
        - hostPath:
            path: /var/lib/kubelet/pods
            type: DirectoryOrCreate
          name: mountpoint-dir
        - hostPath:
            path: /var/lib/kubelet/plugins_registry
            type: Directory
          name: registration-dir
        - hostPath:
            path: /var/lib/kubelet/plugins
            type: Directory
          name: plugins-dir
        - hostPath:
            path: /var/lib/csi-volumes-map/
            type: DirectoryOrCreate
          name: csi-volumes-map
        - emptyDir:
            # this tells Kubernetes to mount a tmpfs (RAM-backed filesystem)
            medium: Memory
          name: csi-data-dir
        - hostPath:
            path: /dev
            type: Directory
          name: dev-dir
`)

func nodeYamlBytes() ([]byte, error) {
	return _nodeYaml, nil
}

func nodeYaml() (*asset, error) {
	bytes, err := nodeYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "node.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _node_saYaml = []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  name: csi-driver-shared-resource-plugin
  namespace: openshift-cluster-csi-drivers`)

func node_saYamlBytes() ([]byte, error) {
	return _node_saYaml, nil
}

func node_saYaml() (*asset, error) {
	bytes, err := node_saYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "node_sa.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacNode_bindingYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: shared-resource-secret-configmap-share-watch-sar-create
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: shared-resource-secret-configmap-share-watch-sar-create
subjects:
  - kind: ServiceAccount
    name: csi-driver-shared-resource-plugin
    namespace: openshift-cluster-csi-drivers`)

func rbacNode_bindingYamlBytes() ([]byte, error) {
	return _rbacNode_bindingYaml, nil
}

func rbacNode_bindingYaml() (*asset, error) {
	bytes, err := rbacNode_bindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/node_binding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacNode_privileged_bindingYaml = []byte(`kind: ClusterRoleBinding
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: shared-resource-node-privileged-binding
subjects:
  - kind: ServiceAccount
    name: csi-driver-shared-resource-plugin
    namespace: openshift-cluster-csi-drivers
roleRef:
  kind: ClusterRole
  name: shared-resource-privileged-role
  apiGroup: rbac.authorization.k8s.io`)

func rbacNode_privileged_bindingYamlBytes() ([]byte, error) {
	return _rbacNode_privileged_bindingYaml, nil
}

func rbacNode_privileged_bindingYaml() (*asset, error) {
	bytes, err := rbacNode_privileged_bindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/node_privileged_binding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacNode_roleYaml = []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: shared-resource-secret-configmap-share-watch-sar-create
rules:
  - apiGroups:
      - ""
    resources:
      - secrets
      - configmaps
      - pods
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - sharedresource.openshift.io
    resources:
      - sharedconfigmaps
      - sharedsecrets
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - authorization.k8s.io
    resources:
      - subjectaccessreviews
    verbs:
      - create`)

func rbacNode_roleYamlBytes() ([]byte, error) {
	return _rbacNode_roleYaml, nil
}

func rbacNode_roleYaml() (*asset, error) {
	bytes, err := rbacNode_roleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/node_role.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacPrivileged_roleYaml = []byte(`
# TODO: create custom SCC with things that the CSI driver needs
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: shared-resource-privileged-role
rules:
  - apiGroups: ["security.openshift.io"]
    resourceNames: ["privileged"]
    resources: ["securitycontextconstraints"]
    verbs: ["use"]`)

func rbacPrivileged_roleYamlBytes() ([]byte, error) {
	return _rbacPrivileged_roleYaml, nil
}

func rbacPrivileged_roleYaml() (*asset, error) {
	bytes, err := rbacPrivileged_roleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/privileged_role.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacPrometheus_roleYaml = []byte(`# Role for accessing metrics exposed by the operator
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: shared-resource-prometheus
  namespace: openshift-cluster-csi-drivers
rules:
  - apiGroups:
      - ""
    resources:
      - services
      - endpoints
      - pods
    verbs:
      - get
      - list
      - watch`)

func rbacPrometheus_roleYamlBytes() ([]byte, error) {
	return _rbacPrometheus_roleYaml, nil
}

func rbacPrometheus_roleYaml() (*asset, error) {
	bytes, err := rbacPrometheus_roleYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/prometheus_role.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _rbacPrometheus_rolebindingYaml = []byte(`# Grant cluster-monitoring access to the operator metrics service
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: shared-resource-prometheus
  namespace: openshift-cluster-csi-drivers
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: shared-resource-prometheus
subjects:
  - kind: ServiceAccount
    name: prometheus-k8s
    namespace: openshift-monitoring

`)

func rbacPrometheus_rolebindingYamlBytes() ([]byte, error) {
	return _rbacPrometheus_rolebindingYaml, nil
}

func rbacPrometheus_rolebindingYaml() (*asset, error) {
	bytes, err := rbacPrometheus_rolebindingYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "rbac/prometheus_rolebinding.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _serviceYaml = []byte(`kind: Service
apiVersion: v1
metadata:
  name: shared-resource-csi-driver-node
  namespace: openshift-cluster-csi-drivers
  labels:
    app: shared-resource-csi-driver-node
spec:
  selector:
    app: shared-resource-csi-driver-node
  ports:
    - name: dummy
      port: 12345`)

func serviceYamlBytes() ([]byte, error) {
	return _serviceYaml, nil
}

func serviceYaml() (*asset, error) {
	bytes, err := serviceYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "service.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

var _servicemonitorYaml = []byte(`---
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: shared-resource-csi-driver-node
  namespace: openshift-cluster-csi-drivers
  annotations:
    include.release.openshift.io/ibm-cloud-managed: "true"
    include.release.openshift.io/self-managed-high-availability: "true"
    include.release.openshift.io/single-node-developer: "true"
spec:
  endpoints:
    - port: metrics
  selector:
    matchLabels:
      app: shared-resource-csi-driver-node
`)

func servicemonitorYamlBytes() ([]byte, error) {
	return _servicemonitorYaml, nil
}

func servicemonitorYaml() (*asset, error) {
	bytes, err := servicemonitorYamlBytes()
	if err != nil {
		return nil, err
	}

	info := bindataFileInfo{name: "servicemonitor.yaml", size: 0, mode: os.FileMode(0), modTime: time.Unix(0, 0)}
	a := &asset{bytes: bytes, info: info}
	return a, nil
}

// Asset loads and returns the asset for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func Asset(name string) ([]byte, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("Asset %s can't read by error: %v", name, err)
		}
		return a.bytes, nil
	}
	return nil, fmt.Errorf("Asset %s not found", name)
}

// MustAsset is like Asset but panics when Asset would return an error.
// It simplifies safe initialization of global variables.
func MustAsset(name string) []byte {
	a, err := Asset(name)
	if err != nil {
		panic("asset: Asset(" + name + "): " + err.Error())
	}

	return a
}

// AssetInfo loads and returns the asset info for the given name.
// It returns an error if the asset could not be found or
// could not be loaded.
func AssetInfo(name string) (os.FileInfo, error) {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	if f, ok := _bindata[cannonicalName]; ok {
		a, err := f()
		if err != nil {
			return nil, fmt.Errorf("AssetInfo %s can't read by error: %v", name, err)
		}
		return a.info, nil
	}
	return nil, fmt.Errorf("AssetInfo %s not found", name)
}

// AssetNames returns the names of the assets.
func AssetNames() []string {
	names := make([]string, 0, len(_bindata))
	for name := range _bindata {
		names = append(names, name)
	}
	return names
}

// _bindata is a table, holding each asset generator, mapped to its name.
var _bindata = map[string]func() (*asset, error){
	"csidriver.yaml":                    csidriverYaml,
	"metrics_service.yaml":              metrics_serviceYaml,
	"node.yaml":                         nodeYaml,
	"node_sa.yaml":                      node_saYaml,
	"rbac/node_binding.yaml":            rbacNode_bindingYaml,
	"rbac/node_privileged_binding.yaml": rbacNode_privileged_bindingYaml,
	"rbac/node_role.yaml":               rbacNode_roleYaml,
	"rbac/privileged_role.yaml":         rbacPrivileged_roleYaml,
	"rbac/prometheus_role.yaml":         rbacPrometheus_roleYaml,
	"rbac/prometheus_rolebinding.yaml":  rbacPrometheus_rolebindingYaml,
	"service.yaml":                      serviceYaml,
	"servicemonitor.yaml":               servicemonitorYaml,
}

// AssetDir returns the file names below a certain
// directory embedded in the file by go-bindata.
// For example if you run go-bindata on data/... and data contains the
// following hierarchy:
//     data/
//       foo.txt
//       img/
//         a.png
//         b.png
// then AssetDir("data") would return []string{"foo.txt", "img"}
// AssetDir("data/img") would return []string{"a.png", "b.png"}
// AssetDir("foo.txt") and AssetDir("notexist") would return an error
// AssetDir("") will return []string{"data"}.
func AssetDir(name string) ([]string, error) {
	node := _bintree
	if len(name) != 0 {
		cannonicalName := strings.Replace(name, "\\", "/", -1)
		pathList := strings.Split(cannonicalName, "/")
		for _, p := range pathList {
			node = node.Children[p]
			if node == nil {
				return nil, fmt.Errorf("Asset %s not found", name)
			}
		}
	}
	if node.Func != nil {
		return nil, fmt.Errorf("Asset %s not found", name)
	}
	rv := make([]string, 0, len(node.Children))
	for childName := range node.Children {
		rv = append(rv, childName)
	}
	return rv, nil
}

type bintree struct {
	Func     func() (*asset, error)
	Children map[string]*bintree
}

var _bintree = &bintree{nil, map[string]*bintree{
	"csidriver.yaml":       {csidriverYaml, map[string]*bintree{}},
	"metrics_service.yaml": {metrics_serviceYaml, map[string]*bintree{}},
	"node.yaml":            {nodeYaml, map[string]*bintree{}},
	"node_sa.yaml":         {node_saYaml, map[string]*bintree{}},
	"rbac": {nil, map[string]*bintree{
		"node_binding.yaml":            {rbacNode_bindingYaml, map[string]*bintree{}},
		"node_privileged_binding.yaml": {rbacNode_privileged_bindingYaml, map[string]*bintree{}},
		"node_role.yaml":               {rbacNode_roleYaml, map[string]*bintree{}},
		"privileged_role.yaml":         {rbacPrivileged_roleYaml, map[string]*bintree{}},
		"prometheus_role.yaml":         {rbacPrometheus_roleYaml, map[string]*bintree{}},
		"prometheus_rolebinding.yaml":  {rbacPrometheus_rolebindingYaml, map[string]*bintree{}},
	}},
	"service.yaml":        {serviceYaml, map[string]*bintree{}},
	"servicemonitor.yaml": {servicemonitorYaml, map[string]*bintree{}},
}}

// RestoreAsset restores an asset under the given directory
func RestoreAsset(dir, name string) error {
	data, err := Asset(name)
	if err != nil {
		return err
	}
	info, err := AssetInfo(name)
	if err != nil {
		return err
	}
	err = os.MkdirAll(_filePath(dir, filepath.Dir(name)), os.FileMode(0755))
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(_filePath(dir, name), data, info.Mode())
	if err != nil {
		return err
	}
	err = os.Chtimes(_filePath(dir, name), info.ModTime(), info.ModTime())
	if err != nil {
		return err
	}
	return nil
}

// RestoreAssets restores an asset under the given directory recursively
func RestoreAssets(dir, name string) error {
	children, err := AssetDir(name)
	// File
	if err != nil {
		return RestoreAsset(dir, name)
	}
	// Dir
	for _, child := range children {
		err = RestoreAssets(dir, filepath.Join(name, child))
		if err != nil {
			return err
		}
	}
	return nil
}

func _filePath(dir, name string) string {
	cannonicalName := strings.Replace(name, "\\", "/", -1)
	return filepath.Join(append([]string{dir}, strings.Split(cannonicalName, "/")...)...)
}
