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
RUN wget https://dl.google.com/go/go1.13.linux-amd64.tar.gz && \
    go_sha256=68a2297eb099d1a76097905a2ce334e3155004ec08cdea85f24527be3c48e856 && \
    echo "$go_sha256 go1.13.linux-amd64.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xvzf go1.13.linux-amd64.tar.gz && \
    rm -rf go1.13.linux-amd64.tar.gz

# Install helm/tiller
RUN wget https://get.helm.sh/helm-v2.14.3-linux-amd64.tar.gz && \
    helm_sha256=38614a665859c0f01c9c1d84fa9a5027364f936814d1e47839b05327e400bf55 && \
    echo "$helm_sha256 helm-v2.14.3-linux-amd64.tar.gz" | sha256sum -c - && \
    tar xzvf helm-v2.14.3-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin && mv linux-amd64/tiller /usr/local/bin && \
    rm -rf linux-amd64 helm-v2.14.3-linux-amd64.tar.gz && \
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

ENV PATH=${PATH}:/usr/local/go/bin \
    GOROOT=/usr/local/go \
    GOPATH=/go
