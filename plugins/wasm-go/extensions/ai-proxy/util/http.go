package util

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/wasm-go/pkg/log"
)

const (
	HeaderContentType = "Content-Type"

	MimeTypeTextPlain       = "text/plain"
	MimeTypeApplicationJson = "application/json"
)

var (
	RegRetrieveBatchPath                        = regexp.MustCompile(`^.*/v1/batches/(?P<batch_id>[^/]+)$`)
	RegCancelBatchPath                          = regexp.MustCompile(`^.*/v1/batches/(?P<batch_id>[^/]+)/cancel$`)
	RegRetrieveFilePath                         = regexp.MustCompile(`^.*/v1/files/(?P<file_id>[^/]+)$`)
	RegRetrieveFileContentPath                  = regexp.MustCompile(`^.*/v1/files/(?P<file_id>[^/]+)/content$`)
	RegRetrieveFineTuningJobPath                = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)$`)
	RegRetrieveFineTuningJobEventsPath          = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)/events$`)
	RegRetrieveFineTuningJobCheckpointsPath     = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)/checkpoints$`)
	RegCancelFineTuningJobPath                  = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)/cancel$`)
	RegResumeFineTuningJobPath                  = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)/resume$`)
	RegPauseFineTuningJobPath                   = regexp.MustCompile(`^.*/v1/fine_tuning/jobs/(?P<fine_tuning_job_id>[^/]+)/pause$`)
	RegFineTuningCheckpointPermissionPath       = regexp.MustCompile(`^.*/v1/fine_tuning/checkpoints/(?P<fine_tuned_model_checkpoint>[^/]+)/permissions$`)
	RegDeleteFineTuningCheckpointPermissionPath = regexp.MustCompile(`^.*/v1/fine_tuning/checkpoints/(?P<fine_tuned_model_checkpoint>[^/]+)/permissions/(?P<permission_id>[^/]+)$`)
)

type ErrorHandlerFunc func(statusCodeDetails string, err error) error

var ErrorHandler ErrorHandlerFunc = func(statusCodeDetails string, err error) error {
	return proxywasm.SendHttpResponseWithDetail(500, statusCodeDetails, CreateHeaders(HeaderContentType, MimeTypeTextPlain), []byte(err.Error()), -1)
}

func CreateHeaders(kvs ...string) [][2]string {
	headers := make([][2]string, 0, len(kvs)/2)
	for i := 0; i < len(kvs); i += 2 {
		headers = append(headers, [2]string{kvs[i], kvs[i+1]})
	}
	return headers
}

func OverwriteRequestPath(path string) error {
	if originPath, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		_ = proxywasm.ReplaceHttpRequestHeader("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	return proxywasm.ReplaceHttpRequestHeader(":path", path)
}

func OverwriteRequestAuthorization(credential string) error {
	if exist, _ := proxywasm.GetHttpRequestHeader("X-HI-ORIGINAL-AUTH"); exist == "" {
		if originAuth, err := proxywasm.GetHttpRequestHeader("Authorization"); err == nil {
			_ = proxywasm.AddHttpRequestHeader("X-HI-ORIGINAL-AUTH", originAuth)
		}
	}
	return proxywasm.ReplaceHttpRequestHeader("Authorization", credential)
}

func OverwriteRequestHostHeader(headers http.Header, host string) {
	if originHost, err := proxywasm.GetHttpRequestHeader(":authority"); err == nil {
		headers.Set("X-ENVOY-ORIGINAL-HOST", originHost)
	}
	headers.Set(":authority", host)
}

func OverwriteRequestPathHeader(headers http.Header, path string) {
	if originPath, err := proxywasm.GetHttpRequestHeader(":path"); err == nil {
		headers.Set("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	headers.Set(":path", path)
}

func OverwriteRequestPathHeaderByCapability(headers http.Header, apiName string, mapping map[string]string) {
	mappedPath, exist := mapping[apiName]
	if !exist {
		return
	}
	originPath, err := proxywasm.GetHttpRequestHeader(":path")
	if err == nil {
		headers.Set("X-ENVOY-ORIGINAL-PATH", originPath)
	}
	/**
	这里实现不太优雅，理应通过 apiName 来判断使用哪个正则替换
	但 ApiName 定义在 provider 中， 而 provider 中又引用了 util
	会导致循环引用
	**/
	if strings.Contains(mappedPath, "{") && strings.Contains(mappedPath, "}") {
		replacements := []struct {
			regx *regexp.Regexp
			key  string
		}{
			{RegRetrieveFilePath, "file_id"},
			{RegRetrieveFileContentPath, "file_id"},
			{RegRetrieveBatchPath, "batch_id"},
			{RegCancelBatchPath, "batch_id"},
		}

		for _, r := range replacements {
			if r.regx.MatchString(originPath) {
				subMatch := r.regx.FindStringSubmatch(originPath)
				if subMatch == nil {
					continue
				}
				index := r.regx.SubexpIndex(r.key)
				if index < 0 || index >= len(subMatch) {
					continue
				}
				id := subMatch[index]
				mappedPath = r.regx.ReplaceAllStringFunc(mappedPath, func(s string) string {
					return strings.Replace(s, "{"+r.key+"}", id, 1)
				})
			}
		}
	}
	headers.Set(":path", mappedPath)
	log.Debugf("[OverwriteRequestPath] originPath=%s, mappedPath=%s", originPath, mappedPath)
}

func OverwriteRequestAuthorizationHeader(headers http.Header, credential string) {
	if exist := headers.Get("X-HI-ORIGINAL-AUTH"); exist == "" {
		if originAuth := headers.Get("Authorization"); originAuth != "" {
			headers.Set("X-HI-ORIGINAL-AUTH", originAuth)
		}
	}
	headers.Set("Authorization", credential)
}

func HeaderToSlice(header http.Header) [][2]string {
	slice := make([][2]string, 0, len(header))
	for key, values := range header {
		for _, value := range values {
			slice = append(slice, [2]string{key, value})
		}
	}
	return slice
}

func SliceToHeader(slice [][2]string) http.Header {
	header := make(http.Header)
	for _, pair := range slice {
		key := pair[0]
		value := pair[1]
		header.Add(key, value)
	}
	return header
}

func GetOriginalRequestHeaders() http.Header {
	originalHeaders, _ := proxywasm.GetHttpRequestHeaders()
	return SliceToHeader(originalHeaders)
}

func GetOriginalResponseHeaders() http.Header {
	originalHeaders, _ := proxywasm.GetHttpResponseHeaders()
	return SliceToHeader(originalHeaders)
}

func ReplaceRequestHeaders(headers http.Header) {
	modifiedHeaders := HeaderToSlice(headers)
	_ = proxywasm.ReplaceHttpRequestHeaders(modifiedHeaders)
}

func ReplaceResponseHeaders(headers http.Header) {
	modifiedHeaders := HeaderToSlice(headers)
	_ = proxywasm.ReplaceHttpResponseHeaders(modifiedHeaders)
}
