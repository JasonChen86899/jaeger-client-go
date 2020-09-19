package jaeger

import (
	"net"
	"strings"
)

const (
	servicesIPsTagKey     = "tag.services.ips"
	servicesIPsBaggageKey = "baggage.services.ips"

	tagKeyHttpStatusCode = "http.status_code"
	tagValueHttpCodeBase = 100

	tagKeyError   = "error"
	tagValueError = 1

	traceErrorTagKey         = "tag.tail_based_sampling.error"
	traceParentErrorTagValue = 1
	traceSelfErrorTagValue   = 0

	traceErrorBaggageKey         = "baggage.tail_based_sampling.error"
	traceParentErrorBaggageValue = "1"
	traceSelfErrorBaggageValue   = "0"
)

func getLocalAddress() (string, error) {
	adds, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range adds {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", nil
}

func baggageServiceIps(span *Span) {
	traceErr := span.baggageItem(traceErrorBaggageKey)
	if traceErr == traceParentErrorBaggageValue {
		span.setBaggageItem(servicesIPsBaggageKey, "")
		return
	}

	ips := span.baggageItem(servicesIPsBaggageKey)
	if localIP, err := getLocalAddress(); err == nil && localIP != "" {
		if ips == "" {
			ips = localIP
		} else {
			ips = strings.Join([]string{ips, localIP}, ",")
		}

		span.setBaggageItem(servicesIPsBaggageKey, ips)
	}
}

func baggageSpanTagParentErr(span *Span) {
	traceErr := span.baggageItem(traceErrorBaggageKey)

	if traceErr == traceSelfErrorBaggageValue {
		span.setBaggageItem(traceErrorBaggageKey, traceParentErrorBaggageValue)
		return
	}

	if traceErr == traceParentErrorBaggageValue {
		return
	}

	needBaggage := false
	// check self span is special span
	for _, tag := range span.tags {
		key := tag.key
		value := tag.value
		switch key {
		case tagKeyHttpStatusCode:
			if httpCode, ok := value.(int64); ok {
				if httpCode/tagValueHttpCodeBase != 2 {
					needBaggage = true
				}
			}

		case tagKeyError:
			if err, ok := value.(int); ok {
				if err == tagValueError {
					needBaggage = true
				}
			}
		}
	}

	if needBaggage {
		span.setBaggageItem(traceErrorBaggageKey, traceSelfErrorBaggageValue)
	}
}
