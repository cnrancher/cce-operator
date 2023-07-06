FROM registry.suse.com/bci/bci-base:15.4

RUN zypper update -y && \
    zypper in -y -f vim && \
    zypper clean -a && \
    rm -rf /tmp/* /var/tmp/* /usr/share/doc/packages/*

RUN useradd --uid 1007 cce-operator
ENV KUBECONFIG /home/cce-operator/.kube/config
COPY bin/cce-operator /usr/bin/
USER 1007
ENTRYPOINT ["/bin/bash"]
CMD ["cce-operator", "--debug"]
