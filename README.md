#Overview

##Dependencies
  1. Install Golang, see:
https://golang.org/
  2. Go the the top directory and set the GOPATH using:
```
$ export GOPATH="$GOPATH":"$PWD"
```
  3. Get Websockets Lib
```
$ go get code.google.com/p/go.net/websocket
```

##Start the Daemon
```
$ cd daemon
$ python server.py
```

##Build and Run the Monitor App
```
$ go run monitor.go
```

##Open the Monitor App in your Browser
```
http://localhost:8080/
```