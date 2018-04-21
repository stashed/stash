FROM golang:1.10-stretch

RUN apt-get update && \
        apt-get install -y git python python-pip && \
        pip install git+https://github.com/ellisonbg/antipackage.git#egg=antipackage && \
        pip install pyyaml

COPY pharmer /bin/
COPY kubectl /bin/
