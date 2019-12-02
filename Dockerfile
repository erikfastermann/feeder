FROM golang:1-stretch
RUN go get -u -v github.com/erikfastermann/feeder
RUN go build -o /feeder github.com/erikfastermann/feeder
RUN cp -r $GOPATH/src/github.com/erikfastermann/feeder/template /template
RUN mkdir -p /var/feeder
RUN mkdir -p /var/feeder-keypairs
CMD /feeder ':443' '/var/feeder-keypairs/live/localhost/fullchain.pem' '/var/feeder-keypairs/live/localhost/privkey.pem' '/template/*' '/var/feeder/feeds.db'
