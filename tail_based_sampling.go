package jaeger

import (
	"net"
	"strings"
)

const (
	/********* private tag key and value ********/

	servicesIPsTagKey     = "tag.services.ips"
	servicesIPsBaggageKey = "baggage.services.ips"

	traceErrorTagKey         = "tag.tail_based_sampling.error"
	traceParentErrorTagValue = 1
	traceSelfErrorTagValue   = 0

	traceErrorBaggageKey         = "baggage.tail_based_sampling.error"
	traceParentErrorBaggageValue = "1"
	traceSelfErrorBaggageValue   = "0"

	/********* Open tag key and value ********/

	tagKeyHttpStatusCode = "http.status_code"
	tagValueHttpCodeBase = 100

	tagKeyServiceError   = "error"
	tagValueServiceError = true
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

// BaggageServiceIPs add local ips and downstream server ip to span baggage for distribution，then need put into jaeger span tags
func BaggageServiceIPs(span *Span, downStreamIP string) {
	traceErr := span.BaggageItem(traceErrorBaggageKey)
	if traceErr == traceParentErrorBaggageValue {
		span.SetBaggageItem(servicesIPsBaggageKey, "")
		return
	}

	ips := span.BaggageItem(servicesIPsBaggageKey)
	if localIP, err := getLocalAddress(); err == nil && localIP != "" {
		if ips == "" {
			ips = localIP
		} else {
			ips = strings.Join([]string{ips, localIP}, ",")
		}

		ips = strings.Join([]string{ips, downStreamIP}, ",")
		span.SetBaggageItem(servicesIPsBaggageKey, ips)
	}
}

func SetHttpCodeTag(span *Span, httpCode int) {
	span.SetTag(tagKeyHttpStatusCode, httpCode)
}

func SetServiceErrorTag(span *Span) {
	span.SetTag(tagKeyServiceError, tagValueServiceError)
}

// BaggageSpanTagSelfErr after set http code tag or err tag, you need call this func
func BaggageSpanTagSelfErr(span *Span) {
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

		case tagKeyServiceError:
			if err, ok := value.(bool); ok {
				if err == tagValueServiceError {
					needBaggage = true
				}
			}
		}
	}

	if needBaggage {
		span.setBaggageItem(traceErrorBaggageKey, traceSelfErrorBaggageValue)
	}
}

// BaggageSpanTagParentErr add parent err for all down stream spans of this trace.
// Attention: this function must call before tracer.Inject()
func BaggageSpanTagParentErr(span *Span) {
	traceErr := span.BaggageItem(traceErrorBaggageKey)

	if traceErr == traceParentErrorBaggageValue {
		return
	} else {
		span.SetBaggageItem(traceErrorBaggageKey, traceParentErrorBaggageValue)
	}
}
