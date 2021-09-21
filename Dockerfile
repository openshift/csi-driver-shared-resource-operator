FROM registry.ci.openshift.org/ocp/builder:rhel-8-golang-1.16-openshift-4.10 AS builder
WORKDIR /go/src/github.com/openshift/shared-resources-operator
COPY . .
RUN make

FROM registry.ci.openshift.org/ocp/4.10:base
COPY --from=builder /go/src/github.com/openshift/shared-resources-operator/shared-resources-operator /usr/bin/
ENTRYPOINT ["/usr/bin/shared-resources-operator"]
LABEL io.k8s.display-name="OpenShift Projected Shared Resources Operator" \
	io.k8s.description="The Projected Shared Resources Operator installs and maintains Projected Shared Resources CSI Driver on a cluster."