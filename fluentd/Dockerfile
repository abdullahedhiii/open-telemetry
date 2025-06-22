FROM fluent/fluentd:v1.16-debian-1

USER root

RUN gem update --system --no-document

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      build-essential \
      ruby-dev \
      libffi-dev \
      libyaml-dev \
      curl \
    && rm -rf /var/lib/apt/lists/*

RUN gem uninstall fluent-plugin-elasticsearch -a -x || true
RUN gem uninstall elasticsearch -a -x || true
RUN gem uninstall elasticsearch-api -a -x || true
RUN gem uninstall elasticsearch-transport -a -x || true

RUN gem install elasticsearch --version "~> 7.17" --no-document

RUN gem install fluent-plugin-elasticsearch -v 5.4.3 --no-document

RUN mkdir -p /fluentd/log && chown -R fluent /fluentd

USER fluent

COPY fluentd.conf /fluentd/etc/fluentd.conf


