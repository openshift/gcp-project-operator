FROM quay.io/redhat-services-prod/openshift/boilerplate:latest AS builder
ENV OPERATOR_PATH=/gcp-project-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}

# Build
RUN make go-build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.9-1795198179@sha256:08d4c70f1baf6537a0728656cf7942d561e74a35916509d205b6519861266352
ENV OPERATOR_PATH=/gcp-project-operator \
    OPERATOR_BIN=gcp-project-operator

WORKDIR /root/
COPY --from=builder /gcp-project-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="gcp-project-operator" \
    io.openshift.managed.description="This operator will be responsible for managing GCP project and credentials."
ENTRYPOINT ["/usr/local/bin/gcp-project-operator"]
