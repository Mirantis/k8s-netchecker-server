# Copyright 2017 Mirantis
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM mkoppanen/etcdtool

MAINTAINER Artem Roma <aroma@mirantis.com>

COPY _output/server /usr/bin/netchecker-server

ENTRYPOINT ["netchecker-server", "-logtostderr"]
CMD ["-v=5", "-kubeproxyinit", "-endpoint=0.0.0.0:8081"]
