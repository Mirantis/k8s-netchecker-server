FROM busybox

MAINTAINER Artem Roma <aroma@mirantis.com>

COPY _output/server /usr/bin/netchecker-server

ENTRYPOINT ["netchecker-server", "-logtostderr"]
CMD ["-v 5", "-kubeproxyinit", "-entrypoint 0.0.0.0:8081"]
