FROM registry.ci.openshift.org/stolostron/builder:go1.20-linux AS plugin-builder
ENV POLICY_GENERATOR_TAG=v1.12.2

WORKDIR go/src/github.com/open-cluster-management/multicloud-operators-subscription
COPY . .
RUN make -f Makefile build

WORKDIR /policy-generator
RUN git clone --branch=${POLICY_GENERATOR_TAG} --depth=1 \
        https://github.com/open-cluster-management-io/policy-generator-plugin
RUN cd "/policy-generator/policy-generator-plugin" && \
        make build-binary && \
        mv "PolicyGenerator" "/policy-generator/"

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest

RUN  microdnf update -y \ 
        && rpm -e --nodeps tzdata \
        && microdnf install tzdata \
        && microdnf install git-core \
        && microdnf install openssh-clients \
        && microdnf clean all

ENV OPERATOR=/usr/local/bin/multicluster-operators-subscription \
    USER_UID=1001 \
    USER_NAME=multicluster-operators-subscription \
    ZONEINFO=/usr/share/timezone \
    KUSTOMIZE_PLUGIN_HOME=/etc/kustomize/plugin

# install operator binary
COPY --from=plugin-builder go/src/github.com/open-cluster-management/multicloud-operators-subscription/build/_output/bin/multicluster-operators-subscription ${OPERATOR}
COPY --from=plugin-builder go/src/github.com/open-cluster-management/multicloud-operators-subscription/build/_output/bin/multicluster-operators-placementrule /usr/local/bin
COPY --from=plugin-builder go/src/github.com/open-cluster-management/multicloud-operators-subscription/build/_output/bin/uninstall-crd /usr/local/bin
COPY --from=plugin-builder go/src/github.com/open-cluster-management/multicloud-operators-subscription/build/_output/bin/appsubsummary /usr/local/bin

# install the policy generator Kustomize plugin
RUN mkdir -p $KUSTOMIZE_PLUGIN_HOME/policy.open-cluster-management.io/v1/policygenerator
COPY --from=plugin-builder /policy-generator/PolicyGenerator $KUSTOMIZE_PLUGIN_HOME/policy.open-cluster-management.io/v1/policygenerator/PolicyGenerator

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup
     
ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
