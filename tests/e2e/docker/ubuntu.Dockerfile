FROM ubuntu:24.04
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
        curl ca-certificates sudo build-essential \
        nodejs npm python3 python3-pip pipx \
        && rm -rf /var/lib/apt/lists/*
COPY ./pkgr /usr/local/bin/pkgr
COPY ./tests/e2e/scripts/ubuntu_smoke.sh /smoke.sh
RUN chmod +x /usr/local/bin/pkgr /smoke.sh
ENTRYPOINT ["/smoke.sh"]
