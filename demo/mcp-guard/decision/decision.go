package decision

import "strings"

type Config struct {
    AllowedCapabilities       []string
    SubjectPolicy             map[string][]string
    RequestedCapabilityHeader string
    Shadow                    bool
}

type Input struct {
    Headers map[string]string
}

type Result struct {
    Allowed        bool
    Shadow         bool
    Reason         string
    Subject        string
    RequestedCap   string
    EffectiveAllow []string
}

// ExtractSubject tries:
// 1) X-Subject header
// 2) Authorization: Bearer <subject> (demo purpose only)
func ExtractSubject(h map[string]string) string {
    if s := h["X-Subject"]; s != "" {
        return s
    }
    if auth := h["authorization"]; auth != "" {
        if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
            return strings.TrimSpace(auth[7:])
        }
    }
    return ""
}

func toSet(arr []string) map[string]struct{} {
    m := map[string]struct{}{}
    for _, v := range arr {
        m[v] = struct{}{}
    }
    return m
}

func intersect(a, b map[string]struct{}) []string {
    out := []string{}
    for k := range a {
        if _, ok := b[k]; ok {
            out = append(out, k)
        }
    }
    return out
}

func CheckAccess(cfg Config, in Input) Result {
    subject := ExtractSubject(in.Headers)
    reqCap := ""
    if cfg.RequestedCapabilityHeader != "" {
        reqCap = in.Headers[strings.ToLower(cfg.RequestedCapabilityHeader)]
        if reqCap == "" {
            reqCap = in.Headers[cfg.RequestedCapabilityHeader]
        }
    }
    allowedRoute := toSet(cfg.AllowedCapabilities)
    subjCaps := toSet(cfg.SubjectPolicy[subject])
    eff := intersect(allowedRoute, subjCaps)
    r := Result{Shadow: cfg.Shadow, Subject: subject, RequestedCap: reqCap, EffectiveAllow: eff}
    if subject == "" {
        r.Reason = "no-subject"
        return r
    }
    if len(eff) == 0 {
        r.Reason = "no-effective-capability"
        return r
    }
    if reqCap != "" {
        if _, ok := toSet(eff)[reqCap]; !ok {
            r.Reason = "requested-cap-not-allowed"
            return r
        }
    }
    r.Allowed = true
    r.Reason = "ok"
    return r
}

