FROM fedora:40
RUN dnf install -y nodejs npm python3 python3-pip pipx git \
    && dnf clean all
COPY ./pkgr /usr/local/bin/pkgr
COPY ./tests/e2e/scripts/fedora_smoke.sh /smoke.sh
RUN chmod +x /usr/local/bin/pkgr /smoke.sh
ENTRYPOINT ["/smoke.sh"]
