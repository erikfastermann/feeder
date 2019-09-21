FROM golang:1-stretch
RUN go get -u github.com/erikfastermann/feeder
RUN go build -o /feeder github.com/erikfastermann/feeder
RUN cp -r $GOPATH/src/github.com/erikfastermann/feeder/template /template
RUN mkdir -p /var/feeder
CMD /feeder ':80' '/template/*' 'unsafe'  '/var/feeder/feeds.db'
