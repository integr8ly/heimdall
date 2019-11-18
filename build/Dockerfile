FROM registry.access.redhat.com/ubi8/ubi:8.0

ENV OPERATOR=/usr/local/bin/heimdall \
    USER_UID=1001 \
    USER_NAME=heimdall

# install operator binary
COPY build/_output/bin/heimdall ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
