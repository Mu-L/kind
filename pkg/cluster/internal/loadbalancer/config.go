/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package loadbalancer

import (
	"bytes"
	"text/template"

	"sigs.k8s.io/kind/pkg/errors"
)

// ConfigData is supplied to the loadbalancer config template
type ConfigData struct {
	ControlPlanePort int
	BackendServers   map[string]string
	IPv6             bool
}

// DefaultConfigTemplate is the loadbalancer config template
const DefaultConfigTemplate = `# generated by kind
global
  log /dev/log local0
  log /dev/log local1 notice
  daemon
  # limit memory usage to approximately 18 MB
  maxconn 100000

resolvers docker
  parse-resolv-conf

defaults
  log global
  mode tcp
  option dontlognull
  # TODO: tune these
  timeout connect 5000
  timeout client 50000
  timeout server 50000
  # allow to boot despite dns don't resolve backends
  default-server init-addr none

frontend control-plane
  bind *:{{ .ControlPlanePort }}
  {{ if .IPv6 -}}
  bind :::{{ .ControlPlanePort }};
  {{- end }}
  default_backend kube-apiservers

backend kube-apiservers
  option httpchk GET /healthz
  # TODO: we should be verifying (!)
  {{range $server, $address := .BackendServers}}
  server {{ $server }} {{ $address }} check check-ssl verify none resolvers docker resolve-prefer {{ if $.IPv6 -}} ipv6 {{- else -}} ipv4 {{- end }}
  {{- end}}
`

// Config returns a kubeadm config generated from config data, in particular
// the kubernetes version
func Config(data *ConfigData) (config string, err error) {
	t, err := template.New("loadbalancer-config").Parse(DefaultConfigTemplate)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse config template")
	}
	// execute the template
	var buff bytes.Buffer
	err = t.Execute(&buff, data)
	if err != nil {
		return "", errors.Wrap(err, "error executing config template")
	}
	return buff.String(), nil
}
