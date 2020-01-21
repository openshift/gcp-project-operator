FROM docker.io/golang:latest AS builder
ENV OPERATOR_PATH=/go/src/github.com/openshift/gcp-project-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}
RUN make build

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest
ENV OPERATOR_PATH=/go/src/github.com/openshift/gcp-project-operator \
    OPERATOR_BIN=gcp-project-operator

WORKDIR /root/
COPY --from=builder /go/src/github.com/openshift/gcp-project-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="gcp-project-operator" \
      io.openshift.managed.description="This operator will be responsible for managing GCP project and credentials."
