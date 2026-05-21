FROM archlinux:latest
RUN pacman -Syu --noconfirm nodejs npm python python-pip git \
    && pacman -Scc --noconfirm
COPY ./pkgr /usr/local/bin/pkgr
COPY ./tests/e2e/scripts/arch_smoke.sh /smoke.sh
RUN chmod +x /usr/local/bin/pkgr /smoke.sh
ENTRYPOINT ["/smoke.sh"]
