FROM gliderlabs/alpine:3.3

RUN echo http://nl.alpinelinux.org/alpine/v3.4/community >> /etc/apk/repositories
RUN apk add --no-cache git go nfs-utils openssh fuse
RUN git clone https://github.com/restic/restic \
  && cd restic \
  && go run build.go \
  && cp restic /usr/local/bin/
RUN apk del git go

RUN mkdir /mnt/restic

ENV RESTIC_REPOSITORY="/"
ENV RESTIC_PASSWORD=""
ENV RESTIC_TAG=""
ENV SOURCE_PATH = "/"
# By default backup every 6 hours
ENV BACKUP_CRON="* * * * * *"
ENV RESTIC_FORGET_ARGS=""
COPY backup.sh /backup.sh
RUN chmod +x /backup.sh
COPY entry.sh /entry.sh
RUN touch /var/log/cron.log
ENTRYPOINT ["/entry.sh"]