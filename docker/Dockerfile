# Use a  golang alpine as the base image
FROM public.ecr.aws/docker/library/golang:1.22.0-alpine3.19 as go_builder
RUN apk update
RUN apk add make cmake git alpine-sdk

# Setup

# Read arguments
ARG SERVICE
ARG VERSION_TAG
ARG GIT_COMMIT
ARG COMMIT_DATE

# Set env variables
ENV COMMIT_DATE=$COMMIT_DATE
ENV SERVICE=$SERVICE
ENV GIT_COMMIT=$GIT_COMMIT
ENV VERSION_TAG=$VERSION_TAG
RUN echo "building service: ${SERVICE}, version: ${VERSION_TAG}, git commit: ${GIT_COMMIT}, commit date: ${COMMIT_DATE}"

# Set the working directory
WORKDIR /
COPY . .

# Download dependencies
RUN go mod download

RUN GO111MODULES=on \
    go build \
    -ldflags "-w -linkmode external -extldflags '-static' -X 'github.com/ATMackay/eth-proxy/proxy.Version=${VERSION_TAG}' -X 'github.com/ATMackay/eth-proxy/proxy.CommitDate=${COMMIT_DATE}' -X 'github.com/ATMackay/eth-proxy/service.GitCommit=${GIT_COMMIT}'"  \
    -o /build/eth-proxy ./cmd/eth-proxy/

# Create linux svcuser
RUN mkdir /build/etc && \
    echo "svcuser:x:1010:1010::/sbin/nologin:/bin/false" > /build/etc/passwd && \
    echo "macuser:x:501:20::/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "linuxuser:x:1000:1000::/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "root:x:0:0:root:/sbin/nologin:/bin/false" >> /build/etc/passwd && \
    echo "svcgroup:x:1010:svcuser" > /build/etc/group && \
    echo "macgroup:x:20:macuser" >> /build/etc/group && \
    echo "linuxgroup:x:1000:linuxuser" >> /build/etc/group && \
    echo "root:x:0:root" >> /build/etc/group && \
    mkdir /build/config && \
    chown -R 1010:1010 /build/config


############################################################################################################

#SSL certs
FROM alpine as certs
RUN apk add --no-cache ca-certificates

############################################################################################################


# Copy binary to a scratch container. Let's keep our images nice and small!
FROM scratch
COPY --from=go_builder /build .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Set User
USER svcuser
# Expose the port your application will run on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/eth-proxy"]

