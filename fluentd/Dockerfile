FROM fluent/fluentd:v1.16-1

USER root

# install elasticsearch plugin
RUN gem install fluent-plugin-elasticsearch --no-document

# required directory setup
RUN mkdir -p /fluentd/log && chown -R fluent /fluentd

USER fluent
