FROM appscode/base:8.7

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates cron \
  && apt-get purge -y --auto-remove ca-certificates \
  && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /tmp/*

COPY restic /restic
COPY restik /restik

# Setup runit scripts
COPY sv /etc/sv/
RUN ln -s /etc/sv /etc/service

ENTRYPOINT ["/runit.sh"]
