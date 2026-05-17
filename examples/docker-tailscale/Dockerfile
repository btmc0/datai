FROM mcr.microsoft.com/devcontainers/base:debian

# Project-specific tools (add your own here)
RUN apt-get update && apt-get install -y --no-install-recommends \
    fish \
    && rm -rf /var/lib/apt/lists/*

# jump (auto-updated on each container start)
ARG JUMP_VERSION=0.8.0
ADD https://github.com/sting8k/jump/releases/download/v${JUMP_VERSION}/jump_${JUMP_VERSION}_linux_amd64.tar.gz /tmp/jump.tar.gz
RUN tar xzf /tmp/jump.tar.gz -C /usr/local/bin/ jump jumpd && rm /tmp/jump.tar.gz

COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

ENTRYPOINT ["entrypoint.sh"]
