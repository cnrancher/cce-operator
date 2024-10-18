FROM registry.suse.com/bci/bci-base:15.6

RUN zypper update -y && \
    zypper in -y -f vim && \
    zypper clean -a && \
    rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/*

RUN useradd --uid 1007 cce-operator
ENV KUBECONFIG /home/cce-operator/.kube/config
COPY dist/cce-operator_linux_${TARGETARCH}*/ /usr/bin/
USER 1007
ENTRYPOINT ["cce-operator", "--debug"]
