# Use a  golang alpine as the base image
FROM public.ecr.aws/docker/library/golang:1.21.0-alpine3.17 as go_builder
RUN apk update
RUN apk add make cmake git alpine-sdk

# Setup

# Read arguments
ARG DATE
ARG SERVICE
ARG GIT_SHA

# Set env variables
ENV date=$DATE
ENV service_name=$SERVICE
ENV commit_hash=$GIT_SHA
RUN echo "building service: ${service_name}, git commit: ${commit_hash}, commit date: ${date}"

# Set the working directory
WORKDIR /
COPY . .

# Download dependencies
RUN go mod download


RUN GO111MODULES=on \
    go build \
    -ldflags "-w -linkmode external -extldflags '-static' -X 'github.com/ATMackay/eth-proxy/service.buildDate=${build_date}' -X 'github.com/ATMackay/eth-proxy/service.gitCommit=${commit_hash}'"  \
    -o /out/eth-proxy ./cmd/eth-proxy/

# Create linux svcuser
RUN mkdir /out/etc && \
    echo "svcuser:x:1010:1010::/sbin/nologin:/bin/false" > /out/etc/passwd && \
    echo "macuser:x:501:20::/sbin/nologin:/bin/false" >> /out/etc/passwd && \
    echo "linuxuser:x:1000:1000::/sbin/nologin:/bin/false" >> /out/etc/passwd && \
    echo "root:x:0:0:root:/sbin/nologin:/bin/false" >> /out/etc/passwd && \
    echo "svcgroup:x:1010:svcuser" > /out/etc/group && \
    echo "macgroup:x:20:macuser" >> /out/etc/group && \
    echo "linuxgroup:x:1000:linuxuser" >> /out/etc/group && \
    echo "root:x:0:root" >> /out/etc/group && \
    mkdir /out/config && \
    chown -R 1010:1010 /out/config


############################################################################################################

#SSL certs
FROM alpine as certs
RUN apk add --no-cache ca-certificates

############################################################################################################


# Copy binary to a scratch container. Let's keep our images nice and small!
FROM scratch
COPY --from=go_builder /out .
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Set User
USER svcuser
# Expose the port your application will run on
EXPOSE 8080

# Run the binary
ENTRYPOINT ["/eth-proxy"]

