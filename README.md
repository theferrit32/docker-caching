# docker-caching


This project is intended to enable modifying HTTP requests to Unix domain sockets on the fly by modifying values in the requests. The `main.go` file contains code to overwrite Docker registry locations in Docker client HTTP API requests, however `unix_domain_socket_proxy` will execute any function which accepts a `*http.Request` and modifies its fields.

This project uses Go's module functionality.

To build, which will pull dependencies the first time:
```
$ go build
```

To run,

```
$ ./docker-caching
```

The file `main.go` contains the main code, which defines where to look for the existing `docker.sock` file and where to put the proxy `dockerproxy.sock` file.  It also contains a default value for which registry address to inject into docker pull requests.

To test, pull and run the docker registry image on a machine in the background. Just using simple localhost here but it could be bound to a public interface or passed through by a reverse proxy like NGINX.

```
$ docker pull registry
$ docker run -p 5000:5000 --restart always --name registry registry:2
```

The alpine-based registry can interfaced to through its own REST API, to tell it to populate from a high-level registry, however here we will just use a direct push example to populate the registry with an image. We will use `ubuntu:18.04` merely as an example, and because it won't overlap with `alpine` layers in use by the registry.

```
$ docker pull ubuntu:18.04
$ docker tag ubuntu:18.04 localhost:5000/ubuntu:18.04
$ docker push localhost:5000/ubuntu:18.04
```

If you still were attached to or monitoring the standard out of the registry daemon you should have just seen some traffic indicating that something was pushed. A copy of ubuntu:18.04 should now be in our local registry.

Next we delete the copy from our host daemon's storage.

```
$ docker rmi ubuntu:18.04 localhost:5000/ubuntu:18.04
```

Next run the proxy in one shell.
```
$ ./docker-caching
```

And in another shell, issue a command from the docker client to the proxy socket. Here we will request `ubuntu:18.04` but the proxy will redirect this image reference which is from the DockerHub registry, and point it instead at our `localhost:5000` registry.

```
$ docker -D --host unix:///tmp/dockerproxy.sock pull ubuntu:18.04
Using default tag: latest
latest: Pulling from ubuntu
Digest: sha256:be159ff0e12a38fd2208022484bee14412680727ec992680b66cdead1ba76d19
Status: Image is up to date for localhost:5000/ubuntu:18.04
$ docker images
localhost:5000/ubuntu   18.04              47b19964fb50        4 weeks ago         88.1MB
```

Next steps:

- issue a docker tag API request to set the `localhost:5000/ubuntu` repository value for this image back to just `ubuntu` after the pull response from the daemon completes. This enables a client expecting the `ubuntu:18.04` image to exist with that exact name, so future `docker run` or other commands are seamless after this redirection took place.