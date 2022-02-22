FROM quay.io/app-sre/boilerplate:image-v2.1.0 AS builder

# Replace Go 1.16 with 1.17:
RUN \
    rm -rf /usr/local/go && \
    curl -Lo /tmp/tarball https://go.dev/dl/go1.17.7.linux-amd64.tar.gz && \
    echo 02b111284bedbfa35a7e5b74a06082d18632eff824fd144312f6063943d49259 /tmp/tarball | sha256sum -c && \
    tar -C /usr/local -xf /tmp/tarball && \
    rm /tmp/tarball

ENV OPERATOR_PATH=/gcp-project-operator
COPY . ${OPERATOR_PATH}
WORKDIR ${OPERATOR_PATH}
RUN make go-build

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest
ENV OPERATOR_PATH=/gcp-project-operator \
    OPERATOR_BIN=gcp-project-operator

WORKDIR /root/
COPY --from=builder /gcp-project-operator/build/_output/bin/${OPERATOR_BIN} /usr/local/bin/${OPERATOR_BIN}
LABEL io.openshift.managed.name="gcp-project-operator" \
      io.openshift.managed.description="This operator will be responsible for managing GCP project and credentials."
