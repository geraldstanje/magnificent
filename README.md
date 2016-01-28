#Overview

##Dependencies
  1. Golang 1.5
  2. Set the Gopath
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