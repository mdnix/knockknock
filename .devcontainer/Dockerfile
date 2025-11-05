FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Install systemd and basic tools
RUN apt-get update && \
    apt-get install -y \
        systemd \
        systemd-sysv \
        curl \
        wget \
        git \
        build-essential \
        ca-certificates \
        jq \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# Install Go
ENV GO_VERSION=1.25.3
RUN ARCH=$(dpkg --print-architecture) && \
    wget -q https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz && \
    tar -C /usr/local -xzf go${GO_VERSION}.linux-${ARCH}.tar.gz && \
    rm go${GO_VERSION}.linux-${ARCH}.tar.gz
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install ORAS CLI
ENV ORAS_VERSION=1.3.0
RUN ARCH=$(dpkg --print-architecture) && \
    wget -q https://github.com/oras-project/oras/releases/download/v${ORAS_VERSION}/oras_${ORAS_VERSION}_linux_${ARCH}.tar.gz && \
    tar -xzf oras_${ORAS_VERSION}_linux_${ARCH}.tar.gz -C /usr/local/bin oras && \
    rm oras_${ORAS_VERSION}_linux_${ARCH}.tar.gz && \
    chmod +x /usr/local/bin/oras

# Setup systemd
RUN cd /lib/systemd/system/sysinit.target.wants/ && \
    ls | grep -v systemd-tmpfiles-setup | xargs rm -f && \
    rm -f /lib/systemd/system/multi-user.target.wants/* && \
    rm -f /etc/systemd/system/*.wants/* && \
    rm -f /lib/systemd/system/local-fs.target.wants/* && \
    rm -f /lib/systemd/system/sockets.target.wants/*udev* && \
    rm -f /lib/systemd/system/sockets.target.wants/*initctl* && \
    rm -f /lib/systemd/system/basic.target.wants/* && \
    rm -f /lib/systemd/system/anaconda.target.wants/*

# Create workspace
RUN mkdir -p /workspace && \
    mkdir -p /workspaces/.tmp && \
    mkdir -p /opt/testapp/versions && \
    mkdir -p /var/lib/knockknock && \
    mkdir -p /etc/knockknock

ENV GOTMPDIR="/workspaces/.tmp"

VOLUME ["/sys/fs/cgroup"]
WORKDIR /workspace

CMD ["/lib/systemd/systemd"]