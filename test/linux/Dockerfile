# Image for make test-linux: a plain distribution with what lazyai needs at
# run time (ps from procps, tmux) and jq for the end-to-end checks. The
# test binaries are built on the host and mounted at /t.
ARG BASE=ubuntu:24.04
FROM ${BASE}
RUN apt-get update -qq \
    && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq --no-install-recommends procps tmux jq >/dev/null \
    && rm -rf /var/lib/apt/lists/*
RUN useradd -m dev
USER dev
WORKDIR /home/dev
ENV TERM=xterm-256color
