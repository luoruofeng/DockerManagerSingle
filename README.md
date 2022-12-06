# DockerManagerSingle

----

# INTRODUCE
docker manager system provied API to manage single node's docker engine.
there are 3 servers support service for out.  first server is http server to handle docker operations. second is GRPC server handle docker operations too, but this one is handling long connections.third is a proxy for docker container.because of i don't want to export port to outter.you can setting 3 different port for 3 servers by launch parameters or configuration file.

----

# BUILD

## Window
set GOOS=windows
set GOARCH=amd64
go build -o bin/dms-amd64.exe main.go

## Linux
set GOOS=linux
set GOARCH=amd64
go build -o bin/dms-amd64 main.go
chmod +x bin/dms-amd64

----

# RUN
## Run environment
config.yaml and executable file in the same folder.

dms-amd64.exe
## Start parameters
`--http_enable (default value:true)`
`--grpc_enable (default value:true)`
`--http_port (default value:8998)`
`--grpc_port (default value:8997)`
## Example
```
.\dms-amd64.exe --grpc_enable=false --api_port=9999
```
