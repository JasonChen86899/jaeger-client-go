// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package jaeger

import (
	"net"
	"strings"
	"time"

	"github.com/opentracing/opentracing-go"

	j "github.com/uber/jaeger-client-go/thrift-gen/jaeger"
	"github.com/uber/jaeger-client-go/utils"
)

const (
	servicesIPsTagKey     = "tag.services.ips"
	servicesIPsBaggageKey = "baggage.services.ips"
)

// BuildJaegerThrift builds jaeger span based on internal span.
// TODO: (breaking change) move to internal package.
func BuildJaegerThrift(span *Span) *j.Span {
	span.Lock()
	defer span.Unlock()
	startTime := utils.TimeToMicrosecondsSinceEpochInt64(span.startTime)
	duration := span.duration.Nanoseconds() / int64(time.Microsecond)
	// add local ips to span baggage for distribution，then need put into jaeger span tags
	baggageServiceIps(span)
	// build jaeger span
	jaegerSpan := &j.Span{
		TraceIdLow:    int64(span.context.traceID.Low),
		TraceIdHigh:   int64(span.context.traceID.High),
		SpanId:        int64(span.context.spanID),
		ParentSpanId:  int64(span.context.parentID),
		OperationName: span.operationName,
		Flags:         int32(span.context.samplingState.flags()),
		StartTime:     startTime,
		Duration:      duration,
		Tags:          buildTags(span.tags, span.tracer.options.maxTagValueLength, span.BaggageItem(servicesIPsBaggageKey)),
		Logs:          buildLogs(span.logs),
		References:    buildReferences(span.references),
	}
	return jaegerSpan
}

// BuildJaegerProcessThrift creates a thrift Process type.
// TODO: (breaking change) move to internal package.
func BuildJaegerProcessThrift(span *Span) *j.Process {
	span.Lock()
	defer span.Unlock()
	return buildJaegerProcessThrift(span.tracer, span.BaggageItem(servicesIPsBaggageKey))
}

func buildJaegerProcessThrift(tracer *Tracer, ips string) *j.Process {
	process := &j.Process{
		ServiceName: tracer.serviceName,
		Tags:        buildTags(tracer.tags, tracer.options.maxTagValueLength, ips),
	}
	if tracer.process.UUID != "" {
		process.Tags = append(process.Tags, &j.Tag{Key: TracerUUIDTagKey, VStr: &tracer.process.UUID, VType: j.TagType_STRING})
	}
	return process
}

func buildTags(tags []Tag, maxTagValueLength int, ips string) []*j.Tag {
	jTags := make([]*j.Tag, 0, len(tags))
	for _, tag := range tags {
		jTag := buildTag(&tag, maxTagValueLength)
		jTags = append(jTags, jTag)
	}

	// add ips tag
	ipsJTag := buildTag(&Tag{
		key:   servicesIPsTagKey,
		value: ips,
	}, maxTagValueLength)
	jTags = append(jTags, ipsJTag)
	return jTags
}

func buildLogs(logs []opentracing.LogRecord) []*j.Log {
	jLogs := make([]*j.Log, 0, len(logs))
	for _, log := range logs {
		jLog := &j.Log{
			Timestamp: utils.TimeToMicrosecondsSinceEpochInt64(log.Timestamp),
			Fields:    ConvertLogsToJaegerTags(log.Fields),
		}
		jLogs = append(jLogs, jLog)
	}
	return jLogs
}

func buildTag(tag *Tag, maxTagValueLength int) *j.Tag {
	jTag := &j.Tag{Key: tag.key}
	switch value := tag.value.(type) {
	case string:
		vStr := truncateString(value, maxTagValueLength)
		jTag.VStr = &vStr
		jTag.VType = j.TagType_STRING
	case []byte:
		if len(value) > maxTagValueLength {
			value = value[:maxTagValueLength]
		}
		jTag.VBinary = value
		jTag.VType = j.TagType_BINARY
	case int:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int8:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint8:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int16:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint16:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int32:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint32:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case int64:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case uint64:
		vLong := int64(value)
		jTag.VLong = &vLong
		jTag.VType = j.TagType_LONG
	case float32:
		vDouble := float64(value)
		jTag.VDouble = &vDouble
		jTag.VType = j.TagType_DOUBLE
	case float64:
		vDouble := float64(value)
		jTag.VDouble = &vDouble
		jTag.VType = j.TagType_DOUBLE
	case bool:
		vBool := value
		jTag.VBool = &vBool
		jTag.VType = j.TagType_BOOL
	default:
		vStr := truncateString(stringify(value), maxTagValueLength)
		jTag.VStr = &vStr
		jTag.VType = j.TagType_STRING
	}
	return jTag
}

func buildReferences(references []Reference) []*j.SpanRef {
	retMe := make([]*j.SpanRef, 0, len(references))
	for _, ref := range references {
		if ref.Type == opentracing.ChildOfRef {
			retMe = append(retMe, spanRef(ref.Context, j.SpanRefType_CHILD_OF))
		} else if ref.Type == opentracing.FollowsFromRef {
			retMe = append(retMe, spanRef(ref.Context, j.SpanRefType_FOLLOWS_FROM))
		}
	}
	return retMe
}

func spanRef(ctx SpanContext, refType j.SpanRefType) *j.SpanRef {
	return &j.SpanRef{
		RefType:     refType,
		TraceIdLow:  int64(ctx.traceID.Low),
		TraceIdHigh: int64(ctx.traceID.High),
		SpanId:      int64(ctx.spanID),
	}
}

func baggageServiceIps(span *Span) {
	ips := span.BaggageItem(servicesIPsBaggageKey)
	if localIP, err := getLocalAddress(); err == nil && localIP != "" {
		ips = strings.Join([]string{ips, localIP}, ",")
	}
	span.SetBaggageItem(servicesIPsBaggageKey, ips)
}

func getLocalAddress() (string, error) {
	adds, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, address := range adds {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To16() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", nil
}
