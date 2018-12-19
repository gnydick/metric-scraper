FROM golang:1.10


RUN mkdir -p /go/src/github.com/gnydick/metric-scraper/
RUN mkdir -p /go/bin
RUN useradd -d /go/bin appuser
ADD . /go/src/github.com/gnydick/metric-scraper/
WORKDIR /go/src/github.com/gnydick/metric-scraper/
RUN go get
RUN go build -o metric-scraper .
USER appuser
WORKDIR /go/bin
CMD [ "./metric-scraper" ]
