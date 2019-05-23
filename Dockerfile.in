FROM debian:stretch

ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NONINTERACTIVE_SEEN true

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends apt-transport-https ca-certificates curl bzip2

RUN set -x                                                                                                                                                          \
  && curl -fsSL -o restic.bz2 https://github.com/restic/restic/releases/download/v{RESTIC_VER}/restic_{RESTIC_VER}_{ARG_OS}_{ARG_ARCH}.bz2                          \
  && bzip2 -d restic.bz2                                                                                                                                            \
  && chmod 755 restic                                                                                                                                               \
  && curl -fsSL -o restic_{NEW_RESTIC_VER}.bz2 https://github.com/restic/restic/releases/download/v{NEW_RESTIC_VER}/restic_{NEW_RESTIC_VER}_{ARG_OS}_{ARG_ARCH}.bz2 \
  && bzip2 -d restic_{NEW_RESTIC_VER}.bz2                                                                                                                           \
  && chmod 755 restic_{NEW_RESTIC_VER}



FROM {ARG_FROM}

COPY --from=0 /restic /bin/restic
COPY --from=0 /restic_{NEW_RESTIC_VER} /bin/restic_{NEW_RESTIC_VER}
COPY bin/{ARG_OS}_{ARG_ARCH}/{ARG_BIN} /{ARG_BIN}

# This would be nicer as `nobody:nobody` but distroless has no such entries.
USER 65535:65535

ENTRYPOINT ["/{ARG_BIN}"]
