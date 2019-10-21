FROM registry.access.redhat.com/ubi7/ubi-minimal:latest

ENV OPERATOR=/usr/local/bin/gcp-project-operator \
    USER_UID=1001 \
    USER_NAME=gcp-project-operator

# install operator binary
COPY build/_output/bin/gcp-project-operator ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
