FROM golang:1.6-alpine

ADD . /go/src/kubeup.com/hostroutes
RUN cd /go/src/kubeup.com/hostroutes; go-wrapper install


