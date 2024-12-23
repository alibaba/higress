package annotations

import (
	networking "istio.io/api/networking/v1alpha3"
)

const (
	tlsMinVersion = "tls-min-version"
	tlsMaxVersion = "tls-max-version"
)

var (
	_ Parser               = tls{}
	_ TrafficPolicyHandler = tls{}
)

type tls struct{}

func (t tls) Parse(annotations Annotations, config *Ingress, _ *GlobalContext) error {
	if !needTLSConfig(annotations) {
		return nil
	}

	tlsConfig := &TLSConfig{}
	defer func() {
		config.UpstreamTLS = tlsConfig
	}()

	// Parse minimum TLS version
	if minVersion, err := annotations.ParseStringASAP(tlsMinVersion); err == nil {
		tlsConfig.MinVersion = minVersion
	}

	// Parse maximum TLS version
	if maxVersion, err := annotations.ParseStringASAP(tlsMaxVersion); err == nil {
		tlsConfig.MaxVersion = maxVersion
	}

	return nil
}

func (t tls) ApplyTrafficPolicy(trafficPolicy *networking.TrafficPolicy, _ *networking.TrafficPolicy_PortTrafficPolicy, config *Ingress) {
	tlsConfig := config.UpstreamTLS
	if tlsConfig == nil {
		return
	}

	if trafficPolicy.Tls == nil {
		trafficPolicy.Tls = &networking.ClientTLSSettings{}
	}

	// Apply min version
	if tlsConfig.MinVersion != "" {
		trafficPolicy.Tls.MinProtocolVersion = convertTLSVersion(tlsConfig.MinVersion)
	}

	// Apply max version
	if tlsConfig.MaxVersion != "" {
		trafficPolicy.Tls.MaxProtocolVersion = convertTLSVersion(tlsConfig.MaxVersion)
	}
}

func needTLSConfig(annotations Annotations) bool {
	return annotations.HasASAP(tlsMinVersion) || annotations.HasASAP(tlsMaxVersion)
}

// Helper to convert TLS version string to istio enum
func convertTLSVersion(version string) networking.ClientTLSSettings_TLSProtocol {
	switch version {
	case "TLSv1_0":
		return networking.ClientTLSSettings_TLSV1_0
	case "TLSv1_1":
		return networking.ClientTLSSettings_TLSV1_1
	case "TLSv1_2":
		return networking.ClientTLSSettings_TLSV1_2
	case "TLSv1_3":
		return networking.ClientTLSSettings_TLSV1_3
	default:
		return networking.ClientTLSSettings_TLS_AUTO
	}
}
