FROM quay.io/redhat-services-prod/openshift/boilerplate:image-v8.2.0 AS builder
ENV OPERATOR_PATH=/gcp-project-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}

# Build
RUN make go-build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1763362218
ENV OPERATOR_PATH=/gcp-project-operator \
    OPERATOR_BIN=gcp-project-operator

WORKDIR /root/
COPY --from=builder /gcp-project-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="gcp-project-operator" \
    io.openshift.managed.description="This operator will be responsible for managing GCP project and credentials."
ENTRYPOINT ["/usr/local/bin/gcp-project-operator"]
