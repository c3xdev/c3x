package aws_test

import (
	"bytes"
	"context"
	"fmt"
	"hash/crc32"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/c3xdev/c3x/internal/engine"
	awsusage "github.com/c3xdev/c3x/internal/usage/aws"
	"github.com/fxamacker/cbor/v2"
)

// cborSmithyPrefix is the URL path prefix used by AWS services that have
// migrated to the Smithy RPCv2 CBOR protocol (e.g. CloudWatch since
// service/cloudwatch v1.52+). The full path is:
//
//	/service/<ServiceVersion>/operation/<OperationName>
//
// where ServiceVersion is the Smithy-generated service ID and OperationName
// is the API operation. For CloudWatch the ServiceVersion is
// GraniteServiceVersion20100801; tests reference operations as needed.
const cloudWatchSmithyPathPrefix = "/service/GraniteServiceVersion20100801/operation/"

// cborEncMode encodes time.Time as CBOR tag 1 (Unix epoch with fractional
// seconds), which is what AWS Smithy clients expect for timestamp fields.
var cborEncMode cbor.EncMode

func init() {
	em, err := cbor.EncOptions{
		Time:    cbor.TimeUnixDynamic,
		TimeTag: cbor.EncTagRequired,
	}.EncMode()
	if err != nil {
		panic(err)
	}
	cborEncMode = em
}

type estimates struct {
	t     *testing.T
	usage map[string]interface{}
}

func newEstimates(ctx context.Context, t *testing.T, resource *engine.Estimate) estimates {
	u := make(map[string]interface{})
	err := resource.EstimateUsage(ctx, u)
	if err != nil {
		t.Fatalf("Expected %T.EstimateUsage to succeed, got %s", resource, err)
	}

	for _, item := range resource.UsageSchema {
		value := u[item.Key]
		if value == nil {
			continue
		}
		switch item.ValueType {
		case engine.Int64:
			if _, ok := value.(int64); !ok {
				t.Errorf("Expected %T %s of type an int64, got a %T", resource, item.Key, value)
			}
		case engine.String:
			if _, ok := value.(string); !ok {
				t.Errorf("Expected %T %s of type string, got a %T", resource, item.Key, value)
			}
		case engine.Float64:
			if _, ok := value.(float64); !ok {
				t.Errorf("Expected %T %s of type float64, got a %T", resource, item.Key, value)
			}
		case engine.StringArray:
			if _, ok := value.([]string); !ok {
				t.Errorf("Expected %T %s of type []string, got a %T", resource, item.Key, value)
			}
		case engine.SubResourceUsage:
			if _, ok := value.(map[string]interface{}); !ok {
				t.Errorf("Expected %T %s of type map[string]interface{}, got a %T", resource, item.Key, value)
			}
		case engine.KeyValueMap:
			if _, ok := value.(map[string]float64); !ok {
				t.Errorf("Expected %T %s of type map[string]float64, got a %T", resource, item.Key, value)
			}
		default:
			t.Errorf("Unknown UsageItem.ValueType %v", item.ValueType)
		}
	}

	return estimates{
		t:     t,
		usage: u,
	}
}

type stubbedRequest struct {
	fullPath       *string
	pathPrefix     *string
	bodyFragments  []string
	response       []byte
	responseStatus int
	contentType    string
}

func (sr *stubbedRequest) Then(status int, response string) {
	sr.responseStatus = status
	sr.response = []byte(response)
	sr.contentType = ""
}

// ThenCBOR encodes value as CBOR (Smithy RPCv2 CBOR protocol) and configures
// the stub to return it with the correct Content-Type. Use for AWS services
// that have migrated to the rpc-v2-cbor protocol (currently CloudWatch).
func (sr *stubbedRequest) ThenCBOR(status int, value interface{}) {
	body, err := cborEncMode.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("ThenCBOR: marshal failed: %s", err))
	}
	sr.responseStatus = status
	sr.response = body
	sr.contentType = "application/cbor"
}

// OnPathPrefix restricts the stub to match requests whose URL path starts
// with prefix. Combined with body fragments to disambiguate operations.
func (sr *stubbedRequest) OnPathPrefix(prefix string) *stubbedRequest {
	sr.pathPrefix = &prefix
	return sr
}

type stubbedAWS struct {
	t        *testing.T
	server   *httptest.Server
	ctx      context.Context
	requests []*stubbedRequest
}

func (sa *stubbedAWS) writeResponse(w http.ResponseWriter, sr *stubbedRequest) {
	if sr.contentType != "" {
		w.Header().Set("Content-Type", sr.contentType)
	}
	// Smithy RPCv2 CBOR clients (CloudWatch et al.) require this header on
	// successful responses; without it the SDK errors with "unexpected
	// smithy-protocol response header ''".
	if sr.contentType == "application/cbor" {
		w.Header().Set("smithy-protocol", "rpc-v2-cbor")
	}
	hash := crc32.NewIEEE()
	hash.Write(sr.response)
	w.Header().Set("X-Amz-Crc32", fmt.Sprintf("%d", hash.Sum32()))
	w.WriteHeader(sr.responseStatus)

	if _, err := w.Write(sr.response); err != nil {
		sa.t.Fatalf("Cannot write stubbed HTTP response: %s", err)
	}
}

func (sa *stubbedAWS) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	_, _ = buf.ReadFrom(r.Body)
	r.Body.Close()
	body := buf.String()

	for _, sr := range sa.requests {
		match := true

		if sr.fullPath != nil && *sr.fullPath != r.URL.RequestURI() {
			match = false
		}

		if sr.pathPrefix != nil && !strings.HasPrefix(r.URL.Path, *sr.pathPrefix) {
			match = false
		}

		for _, fragment := range sr.bodyFragments {
			match = match && strings.Contains(body, fragment)
		}

		if match {
			sa.writeResponse(w, sr)
			return
		}
	}
	sa.t.Fatalf("received unexpected stubbed AWS call: %s %s %s", r.Method, r.URL, body)
}

func (sa *stubbedAWS) WhenFullPath(path string) *stubbedRequest {
	sr := &stubbedRequest{
		fullPath: &path,
	}
	sa.requests = append(sa.requests, sr)
	return sr
}

func (sa *stubbedAWS) WhenBody(fragments ...string) *stubbedRequest {
	sr := &stubbedRequest{
		bodyFragments: fragments,
	}
	sa.requests = append(sa.requests, sr)
	return sr
}

func (sa *stubbedAWS) Close() {
	sa.server.Close()
}

func stubAWS(t *testing.T) *stubbedAWS {
	stub := &stubbedAWS{
		t:        t,
		requests: make([]*stubbedRequest, 0),
	}
	stub.server = httptest.NewServer(stub)
	stub.ctx = awsusage.WithTestEndpoint(context.TODO(), stub.server.URL)
	return stub
}
