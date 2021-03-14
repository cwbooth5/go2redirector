FROM ubuntu:focal

# This container creates a golang build environment and builds
# the go2redirector executable. Then it runs it.
# (from the project root) docker build . -t go2redirector
# docker run go2redirector
# Consider running it with limited capabilities.

RUN apt-get update
RUN apt-get install -y wget jq
WORKDIR /tmp
RUN wget https://golang.org/dl/go1.16.linux-amd64.tar.gz
RUN tar -C /usr/local -xzf go1.16.linux-amd64.tar.gz
ENV PATH=$PATH:/usr/local/go/bin

RUN groupadd -r gouser && useradd -r -g gouser gouser
RUN mkdir -p /home/gouser
RUN chown -R gouser:gouser /home/gouser
USER gouser
WORKDIR /home/gouser
RUN mkdir -p api core http static templates

COPY main.go install.sh godb.json.init go.mod go.sum README.md /home/gouser/
COPY api/ /home/gouser/api
COPY static /home/gouser/static
COPY http/ /home/gouser/http
COPY core/ /home/gouser/core
COPY templates/ /home/gouser/templates

RUN go build
RUN ./install.sh

EXPOSE 8080

CMD ["/home/gouser/go2redirector", "-l", "0.0.0.0"]
