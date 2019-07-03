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
RUN wget https://dl.google.com/go/go1.12.6.linux-amd64.tar.gz && \
    go_sha256=dbcf71a3c1ea53b8d54ef1b48c85a39a6c9a935d01fc8291ff2b92028e59913c && \
    echo "$go_sha256 go1.12.6.linux-amd64.tar.gz" | sha256sum -c - && \
    tar -C /usr/local -xvzf go1.12.6.linux-amd64.tar.gz && \
    rm -rf go1.12.6.linux-amd64.tar.gz

# Install helm/tiller
RUN wget https://get.helm.sh/helm-v2.14.1-linux-amd64.tar.gz && \
    helm_sha256=804f745e6884435ef1343f4de8940f9db64f935cd9a55ad3d9153d064b7f5896 && \
    echo "$helm_sha256 helm-v2.14.1-linux-amd64.tar.gz" | sha256sum -c - && \
    tar xzvf helm-v2.14.1-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin && mv linux-amd64/tiller /usr/local/bin && \
    rm -rf linux-amd64 helm-v2.14.1-linux-amd64.tar.gz && \
    helm init --client-only

# Install kubectl
RUN apt-get update && apt-get install -y apt-transport-https && \
    curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | tee -a /etc/apt/sources.list.d/kubernetes.list && \
    apt-get update && \
    apt-get install -y kubectl

# Install node
RUN curl -sL https://deb.nodesource.com/setup_10.x | bash - && \
    apt-get install -y nodejs

ENV PATH=${PATH}:/usr/local/go/bin \
    GOROOT=/usr/local/go \
    GOPATH=/go