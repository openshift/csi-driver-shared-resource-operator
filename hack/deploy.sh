#! /bin/bash

set -e
set -o pipefail

# Deploy csi-driver-shared-resource-operator to the cluster
#
# Deployment can be tuned using the following environment variables:
#
# - OPERATOR_IMAGE: the image for the operator to deploy
# - DRIVER_IMAGE: the image for the CSI driver
# - NODE_REGISTRAR_IMAGE: the image for the csi node registrar
# - LOG_LEVEL: log level for the operator

rm -rf _deploy
mkdir -p _deploy
cp -r manifests/* _deploy/

operatorImage=${OPERATOR_IMAGE:-quay.io/openshift/origin-csi-driver-shared-resource-operator:latest}
driverImage=${DRIVER_IMAGE:-quay.io/openshift/origin-csi-driver-shared-resource:latest}
nodeRegistrar=${NODE_REGISTRAR_IMAGE:-quay.io/openshift/origin-csi-node-driver-registrar:latest}
logLevel=${LOG_LEVEL:-5}

echo "Deploying operator image ${operatorImage}"
echo "Deploying driver image ${driverImage}"
echo "Deploying node registrar image ${nodeRegistrar}"
echo "Using log level ${logLevel}"

sed -i -e "s|\${OPERATOR_IMAGE}|${operatorImage}|g" \
  -e "s|\${DRIVER_IMAGE}|${driverImage}|g" \
  -e "s|\${NODE_DRIVER_REGISTRAR_IMAGE}|${nodeRegistrar}|g" \
  -e "s|\${LOG_LEVEL}|${logLevel}|g" \
  _deploy/12_deployment.yaml

oc apply -f _deploy/
