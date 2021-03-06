/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package trait

import (
	"fmt"
	"strconv"
	"strings"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// The Jolokia trait activates and configures the Jolokia Java agent.
//
// See https://jolokia.org/reference/html/agents.html
//
// +camel-k:trait=jolokia
type jolokiaTrait struct {
	BaseTrait `property:",squash"`
	// The PEM encoded CA certification file path, used to verify client certificates,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt` for OpenShift).
	CaCert *string `property:"ca-cert"`
	// The principal(s) which must be given in a client certificate to allow access to the Jolokia endpoint,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `clientPrincipal=cn=system:master-proxy`, `cn=hawtio-online.hawtio.svc` and `cn=fuse-console.fuse.svc` for OpenShift).
	ClientPrincipal []string `property:"client-principal"`
	// Listen for multicast requests (default `false`)
	DiscoveryEnabled *bool `property:"discovery-enabled"`
	// Mandate the client certificate contains a client flag in the extended key usage section,
	// applicable when `protocol` is `https` and `use-ssl-client-authentication` is `true`
	// (default `true` for OpenShift).
	ExtendedClientCheck *bool `property:"extended-client-check"`
	// The Host address to which the Jolokia agent should bind to. If `"\*"` or `"0.0.0.0"` is given,
	// the servers binds to every network interface (default `"*"`).
	Host *string `property:"host"`
	// The password used for authentication, applicable when the `user` option is set.
	Password *string `property:"password"`
	// The Jolokia endpoint port (default `8778`).
	Port int `property:"port"`
	// The protocol to use, either `http` or `https` (default `https` for OpenShift)
	Protocol *string `property:"protocol"`
	// The user to be used for authentication
	User *string `property:"user"`
	// Whether client certificates should be used for authentication (default `true` for OpenShift).
	UseSslClientAuthentication *bool `property:"use-ssl-client-authentication"`

	// A comma-separated list of additional Jolokia options as defined
	// in https://jolokia.org/reference/html/agents.html#agent-jvm-config[JVM agent configuration options],
	// e.g.: `keystore=...,executor=...`
	Options *string `property:"options"`
}

func newJolokiaTrait() *jolokiaTrait {
	return &jolokiaTrait{
		BaseTrait: newBaseTrait("jolokia"),
		Port:      8778,
	}
}

func (t *jolokiaTrait) Configure(e *Environment) (bool, error) {
	return t.Enabled != nil && *t.Enabled && e.IntegrationInPhase(
		v1.IntegrationPhaseInitialization,
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *jolokiaTrait) Apply(e *Environment) (err error) {
	if e.IntegrationInPhase(v1.IntegrationPhaseInitialization) {
		// Add the Camel management and Jolokia agent dependencies
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.apache.camel/camel-management")
		// TODO: We may want to make the Jolokia version configurable
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "mvn:org.jolokia/jolokia-jvm:jar:agent:1.6.2")
		return nil
	}

	container := e.getIntegrationContainer()
	if container == nil {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionJolokiaAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionContainerNotAvailableReason,
			"",
		)
		return nil
	}

	// Configure the Jolokia Java agent, first with the extra options
	options, err := parseCsvMap(t.Options)
	if err != nil {
		return err
	}

	if len(t.ClientPrincipal) == 1 {
		// Work-around the lack of proper support for multi-valued trait options from the CLI
		t.ClientPrincipal = strings.Split(t.ClientPrincipal[0], ",")
	}

	setDefaultJolokiaOption(options, &t.Host, "host", "*")
	setDefaultJolokiaOption(options, &t.DiscoveryEnabled, "discoveryEnabled", false)

	// Configure HTTPS by default for OpenShift
	if e.DetermineProfile() == v1.TraitProfileOpenShift {
		setDefaultJolokiaOption(options, &t.Protocol, "protocol", "https")
		setDefaultJolokiaOption(options, &t.CaCert, "caCert", "/var/run/secrets/kubernetes.io/serviceaccount/service-ca.crt")
		setDefaultJolokiaOption(options, &t.ExtendedClientCheck, "extendedClientCheck", true)
		setDefaultJolokiaOption(options, &t.UseSslClientAuthentication, "useSslClientAuthentication", true)
		setDefaultJolokiaOption(options, &t.ClientPrincipal, "clientPrincipal", []string{
			// Master API proxy for OpenShift 3
			"cn=system:master-proxy",
			// Default Hawtio and Fuse consoles for OpenShift 4
			"cn=hawtio-online.hawtio.svc",
			"cn=fuse-console.fuse.svc",
		})
	}

	// Then add explicitly set trait configuration properties
	addToJolokiaOptions(options, "caCert", t.CaCert)
	addToJolokiaOptions(options, "clientPrincipal", t.ClientPrincipal)
	addToJolokiaOptions(options, "discoveryEnabled", t.DiscoveryEnabled)
	addToJolokiaOptions(options, "extendedClientCheck", t.ExtendedClientCheck)
	addToJolokiaOptions(options, "host", t.Host)
	addToJolokiaOptions(options, "password", t.Password)
	addToJolokiaOptions(options, "port", t.Port)
	addToJolokiaOptions(options, "protocol", t.Protocol)
	addToJolokiaOptions(options, "user", t.User)
	addToJolokiaOptions(options, "useSslClientAuthentication", t.UseSslClientAuthentication)

	// Options must be sorted so that the environment variable value is consistent over iterations,
	// otherwise the value changes which results in triggering a new deployment.
	optionValues := make([]string, len(options))
	for i, k := range util.SortedStringMapKeys(options) {
		optionValues[i] = k + "=" + options[k]
	}

	container.Args = append(container.Args, "-javaagent:dependencies/org.jolokia.jolokia-jvm-1.6.2-agent.jar="+strings.Join(optionValues, ","))

	containerPort := corev1.ContainerPort{
		Name:          "jolokia",
		ContainerPort: int32(t.Port),
		Protocol:      corev1.ProtocolTCP,
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionJolokiaAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionJolokiaAvailableReason,
		fmt.Sprintf("%s(%s/%d)", container.Name, containerPort.Name, containerPort.ContainerPort),
	)

	container.Ports = append(container.Ports, containerPort)

	return nil
}

func setDefaultJolokiaOption(options map[string]string, option interface{}, key string, value interface{}) {
	// Do not override existing option
	if _, ok := options[key]; ok {
		return
	}
	switch o := option.(type) {
	case **bool:
		if *o == nil {
			v := value.(bool)
			*o = &v
		}
	case **int:
		if *o == nil {
			v := value.(int)
			*o = &v
		}
	case **string:
		if *o == nil {
			v := value.(string)
			*o = &v
		}
	case *[]string:
		if len(*o) == 0 {
			*o = value.([]string)
		}
	}
}

func addToJolokiaOptions(options map[string]string, key string, value interface{}) {
	switch v := value.(type) {
	case *bool:
		if v != nil {
			options[key] = strconv.FormatBool(*v)
		}
	case *int:
		if v != nil {
			options[key] = strconv.Itoa(*v)
		}
	case int:
		options[key] = strconv.Itoa(v)
	case *string:
		if v != nil {
			options[key] = *v
		}
	case string:
		if v != "" {
			options[key] = v
		}
	case []string:
		if len(v) == 1 {
			options[key] = v[0]
		} else {
			for i, vi := range v {
				options[key+"."+strconv.Itoa(i+1)] = vi
			}
		}
	}
}
