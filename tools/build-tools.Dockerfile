FROM ubuntu:18.04

# Install prerequisite
RUN apt-get update && \
    apt-get install -y wget curl build-essential git

# Install azure-cli
RUN apt-get install apt-transport-https lsb-release software-properties-common dirmngr -y && \
    AZ_REPO=$(lsb_release -cs) && \
    echo "deb [arch=amd64] https://packages.microsoft.com/repos/azure-cli/ $AZ_REPO main" | \
        tee /etc/apt/sources.list.d/azure-cli.list && \
    apt-key --keyring /etc/apt/trusted.gpg.d/Microsoft.gpg adv \
        --keyserver packages.microsoft.com \
        --recv-keys BC528686B50D79E339D3721CEB3E94ADBE1229CF && \
    apt-get update && \
    apt-get install -y azure-cli

# Install docker client
RUN wget https://download.docker.com/linux/static/stable/x86_64/docker-19.03.2.tgz && \
    docker_sha256=865038730c79ab48dfed1365ee7627606405c037f46c9ae17c5ec1f487da1375 && \
    echo "$docker_sha256 docker-19.03.2.tgz" | sha256sum -c - && \
    tar xvzf docker-19.03.2.tgz && \
    mv docker/* /usr/local/bin && \
    rm -rf docker docker-19.03.2.tgz

# Install golang
RUN GO_VERSION=1.13.3 && \
    wget https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz && \
    go_sha256=0804bf02020dceaa8a7d7275ee79f7a142f1996bfd0c39216ccb405f93f994c0 && \
    echo "$go_sha256 go${GO_VERSION}.linux-amd64.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xvzf go${GO_VERSION}.linux-amd64.tar.gz && \
    rm -rf go${GO_VERSION}.linux-amd64.tar.gz

# Install helm/tiller
RUN HELM_VERSION=v2.15.2 && \
    wget https://get.helm.sh/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    helm_sha256=a9d2db920bd4b3d824729bbe1ff3fa57ad27760487581af6e5d3156d1b3c2511 && \
    echo "$helm_sha256 helm-${HELM_VERSION}-linux-amd64.tar.gz" | sha256sum -c - && \
    tar xzvf helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin && mv linux-amd64/tiller /usr/local/bin && \
    rm -rf linux-amd64 helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    helm init --client-only

# Install kubectl
RUN apt-get update && apt-get install -y apt-transport-https && \
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list && \
    apt-get update && \
    apt-get install -y kubectl

# Install node
RUN curl -sL https://deb.nodesource.com/setup_12.x | bash - && \
    apt-get install -y nodejs

RUN RELEASE_VERSION=v0.12.0 && \
    curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && \
    curl -LO https://github.com/operator-framework/operator-sdk/releases/download/${RELEASE_VERSION}/operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc && \
    gpg --keyserver keyserver.ubuntu.com --recv-key 0CF50BEE7E4DF6445E08C0EA9AFDE59E90D2B445 && \
    gpg --verify operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu.asc && \
    chmod +x operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu && \
    mkdir -p /usr/local/bin/ && \
    cp operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu /usr/local/bin/operator-sdk && \
    rm operator-sdk-${RELEASE_VERSION}-x86_64-linux-gnu

ENV PATH=${PATH}:/usr/local/go/bin \
    GOROOT=/usr/local/go \
    GOPATH=/go
