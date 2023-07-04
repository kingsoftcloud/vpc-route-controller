package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"k8s.io/klog"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/util"
	"newgit.op.ksyun.com/kce/vpc-route-controller/pkg/util/random"
)

const (
	ENDPOINT = "http://internal.api.ksyun.com"
	CLUSTER  = "cluster"
	NODE     = "node"
	CA       = "ca"
	VROUTE   = "vroute"
	POST     = http.MethodPost
	PUT      = http.MethodPut
	UPDATE   = http.MethodPut
	GET      = http.MethodGet
	DELETE   = http.MethodDelete

	BackOffDuration = 500 * time.Millisecond
	BackOffFactor   = 1.5
	BackOffJitter   = 2.0
	BackOffSteps    = 10
	BackOffCap      = 5 * time.Second
)

var DefaultBackOff = &wait.Backoff{
	Duration: BackOffDuration,
	Factor:   BackOffFactor,
	Jitter:   BackOffJitter,
	Steps:    BackOffSteps,
	Cap:      BackOffCap,
}

type IKopClient interface {
	SetTenantId(value string) IKopClient
	SetInstanceId(value string) IKopClient
	SetRequestIdPrefix(value string) IKopClient
	SetMethod(value string) IKopClient
	SetHeader(value map[string]string) IKopClient
	SetBody(i interface{}) IKopClient
	SetByteBody(value []byte) IKopClient
	SetUrl(value string) IKopClient
	SetUrlQuery(value string, i interface{}) IKopClient
	SetSigner(ServerName, region, AccessKeyId, AccessKeySecret string) IKopClient
	SetBackOff(b *wait.Backoff) IKopClient
	Go() ([]byte, error)
}

type KopClient struct {
	ctx             context.Context
	endpoint        string
	tenantId        string
	instanceId      string
	method          string
	url             string
	requestIdPrefix string
	headers         map[string]string
	body            *bytes.Buffer
	client          *http.Client
	s               *v4.Signer
	region          string
	servername      string
	backoff         *wait.Backoff
}

func NewKopClient(ctx context.Context) *KopClient {
	dataClient := KopClient{
		ctx:             ctx,
		endpoint:        ENDPOINT,
		client:          &http.Client{},
		requestIdPrefix: "vpc-route-controller",
		backoff:         DefaultBackOff,
	}

	return &dataClient
}

func (kop *KopClient) SetEndpoint(value string) IKopClient {
	kop.endpoint = value
	u, err := url.Parse(value)
	if err != nil {
		return kop
	}

	if u.Scheme == "https" {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		kop.client = &http.Client{Transport: tr}
	}
	return kop
}

func (kop *KopClient) SetInstanceId(value string) IKopClient {
	kop.instanceId = value
	return kop
}

func (kop *KopClient) SetTenantId(value string) IKopClient {
	kop.tenantId = value
	return kop
}

func (kop *KopClient) SetRequestIdPrefix(value string) IKopClient {
	if len(value) == 0 {
		value = "vpc-route-controller"
	}

	kop.requestIdPrefix = value
	return kop
}

func (kop *KopClient) SetMethod(value string) IKopClient {
	kop.method = value
	return kop
}

func (kop *KopClient) SetHeader(value map[string]string) IKopClient {
	if kop.headers == nil {
		kop.headers = make(map[string]string)
	}

	for key, val := range value {
		kop.headers[key] = val
	}

	return kop
}

func (kop *KopClient) SetBackOff(backoff *wait.Backoff) IKopClient {
	kop.backoff = backoff
	return kop
}

func (kop *KopClient) SetUrlQuery(value string, i interface{}) IKopClient {
	u := util.ConvertToQueryValues(i)
	if len(value) == 0 {
		kop.url = fmt.Sprintf("%s?%s", kop.endpoint, u.Encode())
	} else {
		kop.url = fmt.Sprintf("%s/%s?%s", kop.endpoint, value, u.Encode())
	}
	return kop
}

func (kop *KopClient) SetBody(i interface{}) IKopClient {
	bodyStr := bytes.NewBuffer(util.ConvertToMap(i))
	kop.body = bodyStr
	return kop
}

func (kop *KopClient) SetByteBody(value []byte) IKopClient {
	bodyStr := bytes.NewBuffer(value)
	kop.body = bodyStr
	return kop
}

func (kop *KopClient) SetUrl(value string) IKopClient {
	kop.url = fmt.Sprintf("%s/%s", kop.endpoint, value)
	return kop
}

func (kop *KopClient) SetSigner(ServerName, region, AccessKeyId, AccessKeySecret string) IKopClient {
	kop.region = region
	kop.servername = ServerName
	kop.s = v4.NewSigner(credentials.NewStaticCredentials(AccessKeyId, AccessKeySecret, ""))
	return kop
}

func (kop *KopClient) Go() (body []byte, err error) {
	if kop.backoff == nil || (kop.backoff.Duration == 0 && kop.backoff.Factor == 0 &&
		kop.backoff.Jitter == 0 && kop.backoff.Steps == 0 && kop.backoff.Cap == 0) {
		kop.backoff = DefaultBackOff
	}

	if kop.ctx == nil {
		kop.ctx = context.Background()
	}
	err = util.RetryWithBackOff(kop.ctx,
		kop.backoff.Duration, kop.backoff.Factor,
		kop.backoff.Jitter, kop.backoff.Steps, kop.backoff.Cap, func() error {
			body, err = kop.send()
			if err != nil {
				klog.Warningf("retry with backoff kop send err: %v", err)
				return err
			}

			return nil
		}, shouldRetry, nil)
	if err != nil {
		klog.Warningf("retry with backoff send err: %v", err)
		return nil, err
	}

	return body, err
}

func (kop *KopClient) send() ([]byte, error) {
	var span opentracing.Span = nil
	klog.V(9).Infof("req url: %s %s body: %v", kop.method, kop.url, kop.body)
	// var body map[string]interface{}
	requ, err := http.NewRequest(kop.method, kop.url, nil)
	xRequestId := kop.genRequestId()

	switch kop.method {
	case POST:
		requ, err = http.NewRequest(kop.method, kop.url, kop.body)
		if err != nil {
			return nil, err
		}
	case UPDATE:
		requ, err = http.NewRequest(kop.method, kop.url, kop.body)
		if err != nil {
			return nil, err
		}
	default:
		requ, err = http.NewRequest(kop.method, kop.url, nil)
		if err != nil {
			return nil, err
		}
	}

	if kop.s != nil {
		if kop.body != nil {
			bodyLen := kop.body.Len()
			if bodyLen > 0 {
				requ.Header.Add("Content-Length", strconv.Itoa(bodyLen))
			}
		}
		body := strings.NewReader(kop.body.String())
		requ, err = http.NewRequest(kop.method, kop.url, body)
		if err != nil {
			klog.Error(err)
			return nil, err
		}
		if _, err := kop.s.Sign(requ, getSeek(body), kop.servername, kop.region, time.Now()); err != nil {
			klog.Error(err)
			return nil, err
		}
	}

	requ.Header.Add("Content-Type", "application/json")
	requ.Header.Add("Accept", "*/*")
	requ.Header.Add("User-Agent", "vpc-route-controller")
	requ.Header.Add("X-Request-ID", xRequestId)
	requ.Header.Add("X-Openstack-Request-Id", xRequestId)

	// define headers
	for k, v := range kop.headers {
		requ.Header.Set(k, v)
	}

	if kop.ctx != nil {
		span, _ = opentracing.StartSpanFromContext(kop.ctx, kop.getServerName())
		span.SetBaggageItem("X-Request-ID", xRequestId)
		defer span.Finish()

		ext.SpanKindRPCClient.Set(span)
		ext.HTTPUrl.Set(span, kop.url)
		ext.HTTPMethod.Set(span, kop.method)
		span.Tracer().Inject(
			span.Context(),
			opentracing.HTTPHeaders,
			opentracing.HTTPHeadersCarrier(requ.Header),
		)
	} else {
		klog.V(9).Infof("context is nil, unable to propagation the tracing info")
	}

	klog.V(9).Infof("req url: %s %s body: %v header %v", kop.method, kop.url, kop.body, requ.Header)
	resp, err := kop.client.Do(requ)
	if err != nil {
		klog.Error(err)
		return nil, err
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if span != nil {
		span.LogKV("http.status_code", resp.StatusCode)
		span.LogKV("http.response", string(data))
	}

	if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 && resp.StatusCode != 204 {
		e := util.ErrorResponse{
			StatusCode: resp.StatusCode,
			Message:    string(data),
		}

		respErr := &util.Error{
			KopError: e,
		}
		klog.Error(respErr.Error())

		return nil, respErr
	}

	defer resp.Body.Close()

	// json.Unmarshal([]byte(data), &body)
	// result := fmt.Sprintln(body[kop.resource])
	// return strings.Replace(result, "\n", "", -1), nil
	return data, nil
}

func (kop *KopClient) getServerName() string {
	if kop.servername != "" {
		return kop.servername
	}

	urlParts := strings.Split(kop.url, "://")
	if len(urlParts) > 1 {
		urlParts = strings.Split(urlParts[1], "/")
		return urlParts[0]
	} else {
		return kop.url
	}
}

type TimeoutError interface {
	error
	Timeout() bool // Is the error a timeout?
}

func shouldRetry(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(TimeoutError)
	if ok {
		return true
	}

	switch err {
	case io.ErrUnexpectedEOF, io.EOF:
		return true
	}
	switch e := err.(type) {
	case *net.DNSError:
		return true
	case *net.OpError:
		switch e.Op {
		case "read", "write":
			return true
		}
	case *url.Error:
		// url.Error can be returned either by net/url if a URL cannot be
		// parsed, or by net/http if the response is closed before the headers
		// are received or parsed correctly. In that later case, e.Op is set to
		// the HTTP method name with the first letter uppercased. We don't want
		// to retry on POST operations, since those are not idempotent, all the
		// other ones should be safe to retry.
		switch e.Op {
		case "Get", "Put", "Delete", "Head":
			return shouldRetry(e.Err)
		default:
			return false
		}
	case *util.Error:
		var ret bool
		if e.KopError.StatusCode == http.StatusBadRequest && strings.Contains(e.KopError.Message, "SecurityTokenExpired") {
			ret = false
		}

		return ret
	}
	return false
}

func (kop *KopClient) genRequestId() string {
	return fmt.Sprintf("%s-%s", kop.requestIdPrefix, generator())
}

func generator() string {
	return random.String(32)
}

func getSeek(body io.Reader) io.ReadSeeker {
	var seeker io.ReadSeeker
	if sr, ok := body.(io.ReadSeeker); ok {
		seeker = sr
	} else {
		seeker = aws.ReadSeekCloser(body)
	}
	return seeker
}
