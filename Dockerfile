FROM ubuntu:14.04
MAINTAINER myname <emailaddress>

RUN apt-get update -y && apt-get install --no-install-recommends -y -q curl build-essential ca-certificates git mercurial bzr
RUN mkdir /goroot && curl https://storage.googleapis.com/golang/go1.3.1.linux-amd64.tar.gz | tar xvzf - -C /goroot --strip-components=1
RUN mkdir /gopath

ENV GOROOT /goroot
ENV GOPATH /gopath
ENV PATH $PATH:$GOROOT/bin:$GOPATH/bin

RUN apt-get install --no-install-recommends -y -q libzmq3-dev pkg-config

WORKDIR /gopath/src/github.com/tedkulp/skypipe/server
ADD . /gopath/src/github.com/tedkulp/skypipe
RUN go get
RUN go build server.go

EXPOSE 9000

ENTRYPOINT ./server
