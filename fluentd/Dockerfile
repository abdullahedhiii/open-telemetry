FROM fluent/fluentd:v1.16-debian-1

USER root

# Update RubyGems itself to ensure proper dependency resolution
RUN gem update --system --no-document

# Install necessary packages
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
      build-essential \
      ruby-dev \
      libffi-dev \
      libyaml-dev \
      curl \
    && rm -rf /var/lib/apt/lists/*

# --- Dependency Resolution Fix ---
# Force uninstall any potentially conflicting older versions
RUN gem uninstall fluent-plugin-elasticsearch -a -x || true
RUN gem uninstall elasticsearch -a -x || true
RUN gem uninstall elasticsearch-api -a -x || true
RUN gem uninstall elasticsearch-transport -a -x || true

# Install compatible versions - the key change is here
# Use elasticsearch 7.x which is more compatible with fluent-plugin-elasticsearch 5.x
RUN gem install elasticsearch --version "~> 7.17" --no-document

# Install a more recent version of fluent-plugin-elasticsearch that fixes the LoadError
# Version 5.4.3 or later has fixes for the transport/connections/selector issue
RUN gem install fluent-plugin-elasticsearch -v 5.4.3 --no-document

# Alternative: If you want the latest stable version, use this instead:
# RUN gem install fluent-plugin-elasticsearch --no-document

# Setup directories
RUN mkdir -p /fluentd/log && chown -R fluent /fluentd

USER fluent

# Copy your Fluentd configuration file
COPY fluentd.conf /fluentd/etc/fluentd.conf

# Uncomment if you want to override the default CMD
# CMD ["fluentd", "-c", "/fluentd/etc/fluentd.conf", "-v"]