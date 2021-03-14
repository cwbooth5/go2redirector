FROM ubuntu:focal as builder

# This container creates a golang build environment and builds
# the go2redirector executable. Then it runs it.
# (from the project root) docker build . -t go2redirector
# docker run --rm -p 8080:8080 go2redirector
# Consider running it with limited capabilities.

RUN apt-get update
RUN apt-get install -y wget jq
WORKDIR /root
RUN wget https://golang.org/dl/go1.16.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.16.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

COPY main.go go.mod go.sum godb.json.init install.sh /root/
COPY api/ /root/api
COPY http/ /root/http
COPY core/ /root/core

RUN GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build
RUN ./install.sh

# This runtime container is much smaller than the build container.
FROM alpine:latest

# needed to get the go binary to run
RUN mkdir /lib64 && ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2

RUN addgroup -S gouser && adduser -S gouser -G gouser
USER gouser
WORKDIR /home/gouser
RUN mkdir -p static templates data
COPY static ./static
COPY templates ./templates
COPY README.md /home/gouser/

# artifacts from the builder container
COPY --from=builder /root/go2redirector .
COPY --from=builder /root/godb.json .
COPY --from=builder /root/go2config.json .

# Move the godb.json file to a volume mount point so it can persist outside the container.
RUN mv /home/gouser/godb.json /home/gouser/data/
RUN ln -s /home/gouser/data/godb.json godb.json

# Normally INFO and ERROR-level stuff goes to this file. Send all to stdout.
RUN ln -sf /dev/stdout /home/gouser/redirector.log

EXPOSE 8080
CMD ["/home/gouser/go2redirector", "-l", "0.0.0.0"]
