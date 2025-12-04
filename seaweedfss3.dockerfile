FROM chrislusf/seaweedfs:4.01

USER root

# Install Python and pip.
RUN apk add --no-cache python3 py3-pip

# Install awscurl via pip.
RUN pip install --no-cache-dir --break-system-packages awscurl
