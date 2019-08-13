FROM ubuntu:18.04 

ENV GOGSTASH_VERSION v1.0.3

ADD https://github.com/pensando/gogstash/releases/download/${GOGSTASH_VERSION}/gogstash.tar.gz /usr/local/bin/

RUN cd /usr/local/bin && tar -xzf gogstash.tar.gz && rm gogstash.tar.gz && chmod +x gogstash

ENTRYPOINT exec gogstash --config /etc/gogstash.yaml
