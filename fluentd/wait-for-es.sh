#!/bin/sh
sleep 5
echo "Waiting for Elasticsearch to be ready..."
until curl -s http://elasticsearch:9200 >/dev/null; do
  echo "Still waiting for Elasticsearch..."
  sleep 2
done
echo "Elasticsearch is up. Starting Fluentd..."
exec fluentd -c /fluentd/etc/fluent.conf -v