# DockerManagerSingle
docker manager system provied API to manage single node's docker engine.
there are 3 servers support service for out.  first server is http server to handle docker operations. second is GRPC server handle docker operations too, but this one is handling long connections.third is a proxy for docker container.because of i don't want to export port to outter.you can setting 3 different port for 3 servers by launch parameters or configuration file.
