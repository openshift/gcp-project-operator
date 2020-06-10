FROM docker.io/golang:latest AS builder

ENV OPERATOR_PATH=/gcp-project-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}
RUN make gobuild

FROM registry.access.redhat.com/ubi7/ubi-minimal:latest
ENV OPERATOR_BIN=gcp-project-operator \
    OPERATOR_PATH=/${OPERATOR_BIN}
ARG  OPERATOR_BASE_PATH=/gcp-project-operator/build/_output/bin

WORKDIR /root/
COPY --from=builder ${OPERATOR_BASE_PATH}/gcp-project-operator /usr/local/bin/gcp-project-operator
COPY --from=builder ${OPERATOR_BASE_PATH}/monitoring /usr/local/bin/monitoring

LABEL io.openshift.managed.name="gcp-project-operator" \
      io.openshift.managed.description="This operator will be responsible for managing GCP project and credentials."
