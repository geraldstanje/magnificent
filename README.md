#Overview

##Dependencies
  1. Golang 1.5
  2. Get Websockets Lib<br>
  ```
  $ go get github.com/gorilla/websocket
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