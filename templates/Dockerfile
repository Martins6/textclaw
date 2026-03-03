FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ENV HOME=/home/user

RUN apt-get update -o Acquire::AllowInsecureRepositories=true -o Acquire::AllowDowngradeToInsecureRepositories=true && apt-get install -y --allow-unauthenticated \
    curl \
    git \
    bash \
    && rm -rf /var/lib/apt/lists/*

RUN curl -fsSL https://opencode.ai/install | bash

RUN useradd -m -s /bin/bash user && \
    mkdir -p /home/user && \
    chown -R user:user /home/user

USER user
WORKDIR /home/user

ENV PATH=/home/user/.opencode/bin:$PATH

CMD ["opencode", "serve", "--port", "8080", "--hostname", "0.0.0.0"]
