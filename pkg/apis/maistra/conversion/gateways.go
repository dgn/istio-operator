package conversion

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v2 "github.com/maistra/istio-operator/pkg/apis/maistra/v2"
	"github.com/maistra/istio-operator/pkg/controller/versions"
)

const (
	imageNameSDS = "node-agent-k8s"
)

var (
	meshExpansionPortsV11 = []corev1.ServicePort{
		{
			Name:       "tcp-pilot-grpc-tls",
			Port:       15011,
			TargetPort: intstr.FromInt(15011),
		},
		{
			Name:       "tcp-mixer-grpc-tls",
			Port:       15004,
			TargetPort: intstr.FromInt(15004),
		},
		{
			Name:       "tcp-citadel-grpc-tls",
			Port:       8060,
			TargetPort: intstr.FromInt(8060),
		},
		{
			Name:       "tcp-dns-tls",
			Port:       853,
			TargetPort: intstr.FromInt(8853),
		},
	}
	meshExpansionPortsV20 = []corev1.ServicePort{
		{
			Name:       "tcp-istiod",
			Port:       15012,
			TargetPort: intstr.FromInt(15012),
		},
		{
			Name:       "tcp-dns-tls",
			Port:       853,
			TargetPort: intstr.FromInt(8853),
		},
	}
)

func populateGatewaysValues(in *v2.ControlPlaneSpec, values map[string]interface{}) error {
	if in.Gateways == nil {
		return nil
	}

	if err := setHelmBoolValue(values, "gateways.enabled", true); err != nil {
		return err
	}

	gateways := in.Gateways

	if gateways.Ingress != nil {
		if gatewayValues, err := gatewayConfigToValues(&gateways.Ingress.GatewayConfig); err == nil {
			if len(gateways.Ingress.MeshExpansionPorts) > 0 {
				untypedSlice := make([]interface{}, len(gateways.Ingress.MeshExpansionPorts))
				for index, port := range gateways.Ingress.MeshExpansionPorts {
					untypedSlice[index] = port
				}
				if portsValue, err := sliceToValues(untypedSlice); err == nil {
					if len(portsValue) > 0 {
						if err := setHelmValue(gatewayValues, "meshExpansionPorts", portsValue); err != nil {
							return err
						}
					}
				} else {
					return err
				}
			}
			if len(gatewayValues) > 0 {
				if err := setHelmValue(values, "gateways.istio-ingressgateway", gatewayValues); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	if gateways.Egress != nil {
		if gatewayValues, err := gatewayConfigToValues(gateways.Egress); err == nil {
			if len(gatewayValues) > 0 {
				if err := setHelmValue(values, "gateways.istio-egressgateway", gatewayValues); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}

	for name, gateway := range gateways.AdditionalGateways {
		if gatewayValues, err := gatewayConfigToValues(&gateway); err == nil {
			if len(gatewayValues) > 0 {
				if err := setHelmValue(values, "gateways."+name, gatewayValues); err != nil {
					return err
				}
			}
		} else {
			return err
		}
	}
	return nil
}

// converts v2.GatewayConfig to values.yaml
func gatewayConfigToValues(in *v2.GatewayConfig) (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if in.Enabled != nil {
		if err := setHelmBoolValue(values, "enabled", *in.Enabled); err != nil {
			return nil, err
		}
	}

	if in.Namespace != "" {
		if err := setHelmStringValue(values, "namespace", in.Namespace); err != nil {
			return nil, err
		}
	}
	if in.RouterMode == "" {
		if err := setHelmStringValue(values, "env.ISTIO_META_ROUTER_MODE", string(v2.RouterModeTypeSNIDNAT)); err != nil {
			return nil, err
		}
	} else {
		if err := setHelmStringValue(values, "env.ISTIO_META_ROUTER_MODE", string(in.RouterMode)); err != nil {
			return nil, err
		}
	}

	if len(in.RequestedNetworkView) > 0 {
		if err := setHelmStringValue(values, "env.ISTIO_META_REQUESTED_NETWORK_VIEW", fmt.Sprintf("\"%s\"", strings.Join(in.RequestedNetworkView, ","))); err != nil {
			return nil, err
		}
	}

	// Service specific settings
	if in.Service.LoadBalancerIP != "" {
		if err := setHelmStringValue(values, "loadBalancerIP", in.Service.LoadBalancerIP); err != nil {
			return nil, err
		}
	}
	if len(in.Service.LoadBalancerSourceRanges) > 0 {
		if err := setHelmStringSliceValue(values, "loadBalancerSourceRanges", in.Service.LoadBalancerSourceRanges); err != nil {
			return nil, err
		}
	}
	if in.Service.ExternalTrafficPolicy != "" {
		if err := setHelmStringValue(values, "externalTrafficPolicy", string(in.Service.ExternalTrafficPolicy)); err != nil {
			return nil, err
		}
	}
	if len(in.Service.ExternalIPs) > 0 {
		if err := setHelmStringSliceValue(values, "externalIPs", in.Service.ExternalIPs); err != nil {
			return nil, err
		}
	}
	if in.Service.Type != "" {
		if err := setHelmStringValue(values, "type", string(in.Service.Type)); err != nil {
			return nil, err
		}
	}
	if len(in.Service.Metadata.Labels) > 0 {
		if err := setHelmStringMapValue(values, "labels", in.Service.Metadata.Labels); err != nil {
			return nil, err
		}
	}
	if len(in.Service.Ports) > 0 {
		untypedSlice := make([]interface{}, len(in.Service.Ports))
		for index, port := range in.Service.Ports {
			untypedSlice[index] = port
		}
		if portsValue, err := sliceToValues(untypedSlice); err == nil {
			if len(portsValue) > 0 {
				if err := setHelmValue(values, "ports", portsValue); err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}

	// gateway SDS
	if in.EnableSDS != nil {
		if err := setHelmBoolValue(values, "sds.enable", *in.EnableSDS); err != nil {
			return nil, err
		}
	}

	// Deployment specific settings
	if in.Runtime != nil {
		runtime := in.Runtime
		if err := populateRuntimeValues(runtime, values); err != nil {
			return nil, err
		}

		if runtime.Pod.Containers != nil {
			// SDS container specific config
			if sdsContainer, ok := runtime.Pod.Containers["ingress-sds"]; ok {
				if sdsContainer.Image != "" {
					if err := setHelmStringValue(values, "sds.image", sdsContainer.Image); err != nil {
						return nil, err
					}
				}
				if sdsContainer.Resources != nil {
					if resourcesValues, err := toValues(sdsContainer.Resources); err == nil {
						if len(resourcesValues) > 0 {
							if err := setHelmValue(values, "sds.resources", resourcesValues); err != nil {
								return nil, err
							}
						}
					} else {
						return nil, err
					}
				}
			}
			// Proxy container specific config
			if proxyContainer, ok := runtime.Pod.Containers["istio-proxy"]; ok {
				if proxyContainer.Resources != nil {
					if resourcesValues, err := toValues(proxyContainer.Resources); err == nil {
						if len(resourcesValues) > 0 {
							if err := setHelmValue(values, "resources", resourcesValues); err != nil {
								return nil, err
							}
						}
					} else {
						return nil, err
					}
				}
			}
		}
	}

	// Additional volumes
	if len(in.Volumes) > 0 {
		configVolumes := make([]map[string]string, 0)
		secretVolumes := make([]map[string]string, 0)
		for _, volume := range in.Volumes {
			if volume.Volume.ConfigMap != nil {
				configVolumes = append(configVolumes, map[string]string{
					"name":          volume.Mount.Name,
					"configMapName": volume.Volume.Name,
					"mountPath":     volume.Mount.MountPath,
				})
			} else if volume.Volume.Secret != nil {
				secretVolumes = append(secretVolumes, map[string]string{
					"name":       volume.Mount.Name,
					"secretName": volume.Volume.Name,
					"mountPath":  volume.Mount.MountPath,
				})
			} else {
				// XXX: ignore misconfigured volumes?
			}
		}
		if len(configVolumes) > 0 {
			if err := setHelmValue(values, "configVolumes", configVolumes); err != nil {
				return nil, err
			}
		}
		if len(secretVolumes) > 0 {
			if err := setHelmValue(values, "secretVolumes", secretVolumes); err != nil {
				return nil, err
			}
		}
	}
	return values, nil
}

func populateGatewaysConfig(in map[string]interface{}, out *v2.GatewaysConfig) error {
	gateways, ok := in["gateways"].(map[string]interface{})
	if ok {
		for name, gateway := range gateways {
			gc := v2.GatewayConfig{}
			gatewayMap, ok := gateway.(map[string]interface{})
			if !ok {
				return fmt.Errorf("Failed to parse gateway.%s: cannot cast to map[string]interface{}", name)
			}

			gatewayValuesToConfig(gatewayMap, &gc)
			// Put it in the correct bucket
			if name == "istio-ingressgateway" {
				igc := &v2.IngressGatewayConfig{
					GatewayConfig: gc,
				}

				// TODO: igc.MeshExpansionPorts
				out.Ingress = igc
			} else if name == "istio-egressgateway" {
				out.Egress = &gc
			} else {
				out.AdditionalGateways[name] = gc
			}

		}
	}
	return nil
}

func gatewayValuesToConfig(in map[string]interface{}, out *v2.GatewayConfig) error {
	out.Enabled = getHelmBoolValue(in, "enabled")
	out.Namespace = getHelmStringValue(in, "namespace")
	out.RequestedNetworkView = strings.Split(getHelmStringValue(in, "env.ISTIO_META_REQUESTED_NETWORK_VIEW"), ",")
	if routerMode := getHelmStringValue(in, "env.ISTIO_META_ROUTER_MODE"); routerMode != "" {
		out.RouterMode = v2.RouterModeType(routerMode)
	} else {
		out.RouterMode = v2.RouterModeTypeSNIDNAT
	}

	out.EnableSDS = getHelmBoolValue(in, "sds.enabled")

	// Service-specific config
	out.Service = v2.GatewayServiceConfig{}
	out.Service.LoadBalancerIP = getHelmStringValue(in, "loadBalancerIP")
	out.Service.LoadBalancerSourceRanges = getHelmStringSliceValue(in, "loadBalancerSourceRanges")
	if externalTrafficPolicy := getHelmStringValue(in, "externalTrafficPolicy"); externalTrafficPolicy != "" {
		out.Service.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyType(externalTrafficPolicy)
	} else {
		out.Service.ExternalTrafficPolicy = corev1.ServiceExternalTrafficPolicyTypeLocal
	}
	out.Service.ExternalIPs = getHelmStringSliceValue(in, "externalIPs")
	out.Service.Metadata.Labels = getHelmStringMapValue(in, "labels")
	return nil
}

func expansionPortsForVersion(version string) ([]corev1.ServicePort, error) {
	switch version {
	case "":
		fallthrough
	case versions.V1_0.String():
		fallthrough
	case versions.V1_1.String():
		return meshExpansionPortsV11, nil
	case versions.V1_2.String():
		return meshExpansionPortsV20, nil
	default:
		return nil, fmt.Errorf("cannot convert for unknown version: %s", version)
	}
}
func addExpansionPorts(in *[]corev1.ServicePort, ports []corev1.ServicePort) {
	portCount := len(*in)
PortsLoop:
	for _, port := range ports {
		for index := 0; index < portCount; index++ {
			if port.Port == (*in)[index].Port {
				continue PortsLoop
			}
			*in = append(*in, port)
		}
	}
}
