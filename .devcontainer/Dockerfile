#-------------------------------------------------------------------------------------------------------------
# Copyright (c) Microsoft Corporation. All rights reserved.
# Licensed under the MIT License. See https://go.microsoft.com/fwlink/?linkid=2090316 for license information.
#-------------------------------------------------------------------------------------------------------------

FROM golang:1.23.5

# Avoid warnings by switching to noninteractive
ENV DEBIAN_FRONTEND=noninteractive

# This Dockerfile adds a non-root 'vscode' user with sudo access. However, for Linux,
# this user's GID/UID must match your local user UID/GID to avoid permission issues
# with bind mounts. Update USER_UID / USER_GID if yours is not 1000. See
# https://aka.ms/vscode-remote/containers/non-root-user for details.
ARG USERNAME=vscode
ARG USER_UID=1000
ARG USER_GID=$USER_UID

# Configure apt, install packages and tools
RUN apt-get update \
    && apt-get -y install --no-install-recommends apt-utils dialog unzip 2>&1 \
    #
    # Verify git, process tools, lsb-release (common in install instructions for CLIs) installed
    && apt-get -y install git iproute2 procps lsb-release \
    #
    # Install Go tools
    && go install github.com/uudashr/gopkgs/v2/cmd/gopkgs@latest \
    && go install github.com/ramya-rao-a/go-outline@latest \
    && go install github.com/acroca/go-symbols@latest \
    && go install github.com/godoctor/godoctor@latest \
    && go install golang.org/x/tools/cmd/gorename@latest \
    && go install github.com/rogpeppe/godef@latest \
    && go install github.com/zmb3/gogetdoc@latest \
    && go install github.com/haya14busa/goplay/cmd/goplay@latest \
    && go install github.com/sqs/goreturns@latest \
    && go install github.com/josharian/impl@latest \
    && go install github.com/davidrjenni/reftools/cmd/fillstruct@latest \
    && go install github.com/fatih/gomodifytags@latest \
    && go install github.com/cweill/gotests/...@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install golang.org/x/lint/golint@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install github.com/mgechev/revive@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install golang.org/x/tools/gopls@latest \
    # Protocol Buffer Compiler
    && PROTOC_VERSION=21.9 \
    && if [ $(dpkg --print-architecture) = "amd64" ]; then PROTOC_ARCH="x86_64"; else PROTOC_ARCH="aarch_64" ; fi \
    && curl -LO "https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-$PROTOC_ARCH.zip" \
    && unzip "protoc-${PROTOC_VERSION}-linux-$PROTOC_ARCH.zip" -d $HOME/.local \
    && mv $HOME/.local/bin/protoc /usr/local/bin/protoc \
    && mv $HOME/.local/include/ /usr/local/bin/include/ \
    && protoc --version \
    #
    # Create a non-root user to use if preferred - see https://aka.ms/vscode-remote/containers/non-root-user.
    && groupadd --gid $USER_GID $USERNAME \
    && useradd -s /bin/bash --uid $USER_UID --gid $USER_GID -m $USERNAME \
    # [Optional] Add sudo support
    && apt-get install -y sudo \
    && echo $USERNAME ALL=\(root\) NOPASSWD:ALL > /etc/sudoers.d/$USERNAME \
    && chmod 0440 /etc/sudoers.d/$USERNAME \
    # Docker install https://docs.docker.com/engine/install/debian/
    && sudo install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/debian/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/docker.gpg \
    && sudo chmod a+r /etc/apt/keyrings/docker.gpg \
    && echo \
      "deb [arch="$(dpkg --print-architecture)" signed-by=/etc/apt/keyrings/docker.gpg] https://download.docker.com/linux/debian \
      "$(. /etc/os-release && echo "$VERSION_CODENAME")" stable" | \
      sudo tee /etc/apt/sources.list.d/docker.list > /dev/null \
    && sudo apt-get update \
    && apt-get install -y docker-ce-cli \
    #
    # Install pip & pre-commit
    && apt-get -y install python3-pip \
    && python3 -m pip install --no-cache-dir --break-system-packages pre-commit \
    #
    # Clean up
    && apt-get autoremove -y \
    && apt-get clean -y \
    && rm -rf /var/lib/apt/lists/*

ENV OPERATOR_RELEASE_VERSION=v1.26.0
RUN ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac) \
    && OS=$(uname | awk '{print tolower($0)}') \
    && OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/${OPERATOR_RELEASE_VERSION} \
    && curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH} \
    && chmod +x operator-sdk_${OS}_${ARCH} \
    && mkdir -p /usr/local/bin/ \
    && cp operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk \
    && rm operator-sdk_${OS}_${ARCH}
