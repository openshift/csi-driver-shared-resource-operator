# csi-driver-shared-resource-operator

An operator to deploy the [Shared Resource CSI Driver](https://github.com/openshift/csi-driver-shared-resource) in OpenShift.

This operator will eventually be installed by the [cluster-storage-operator](https://github.com/openshift/cluster-storage-operator).

NOTE:  at the moment, using this driver is only supported via cloning this repository and executing the commands detailed below.

# Quick start

Before running the operator manually, you must remove the operator installed by CSO/CVO

```shell
# Scale down CVO and CSO
oc scale --replicas=0 deploy/cluster-version-operator -n openshift-cluster-version
oc scale --replicas=0 deploy/cluster-storage-operator -n openshift-cluster-storage-operator

# Delete operator resources from a clone of this repository
oc delete -F -f ./assets
```

To build run `make build`.

To deploy run `make deploy`.  You can override the images used for the CSI Node Driver Registrar, the image for this operator,
and the image used for the Shared Resource CSI Driver that this operator deploys, all via environment variables:
- `NODE_DRIVER_REGISTRAR_IMAGE` where the default is quay.io/openshift/origin-csi-node-driver-registrar:latest
- `OPERATOR_IMAGE` where the default is quay.io/openshift/origin-csi-driver-shared-resource-operator:latest
- `DRIVER_IMAGE`  where the default is quay.io/openshift/origin-csi-driver-shared-resource:latest

