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
RUN wget https://download.docker.com/linux/static/stable/x86_64/docker-18.09.3.tgz && \
    docker_sha256=8b886106cfc362f1043debfe178c35b6f73ec42380b034a3919a235fe331e053 && \
    echo "$docker_sha256 docker-18.09.3.tgz" | sha256sum -c - && \
    tar xvzf docker-18.09.3.tgz && \
    mv docker/* /usr/local/bin && \
    rm -rf docker docker-18.09.3.tgz

# Install golang
RUN wget https://dl.google.com/go/go1.12.linux-amd64.tar.gz && \
    go_sha256=750a07fef8579ae4839458701f4df690e0b20b8bcce33b437e4df89c451b6f13 && \
    echo "$go_sha256 go1.12.linux-amd64.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xvzf go1.12.linux-amd64.tar.gz && \
    rm -rf go1.12.linux-amd64.tar.gz

# Install helm/tiller
RUN wget https://storage.googleapis.com/kubernetes-helm/helm-v2.13.0-linux-amd64.tar.gz && \
    helm_sha256=15eca6ad225a8279de80c7ced42305e24bc5ac60bb7d96f2d2fa4af86e02c794 && \
    echo "$helm_sha256 helm-v2.13.0-linux-amd64.tar.gz" | sha256sum -c - && \
    tar xzvf helm-v2.13.0-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin && mv linux-amd64/tiller /usr/local/bin && \
    rm -rf linux-amd64 helm-v2.13.0-linux-amd64.tar.gz && \
    helm init --client-only

ENV PATH=${PATH}:/usr/local/go/bin \
    GOROOT=/usr/local/go \
    GOPATH=/go