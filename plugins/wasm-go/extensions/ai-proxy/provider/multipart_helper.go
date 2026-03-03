package provider

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
)

type multipartImageRequest struct {
	Model        string
	Prompt       string
	Size         string
	OutputFormat string
	N            int
	ImageURLs    []string
	HasMask      bool
}

func isMultipartFormData(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.EqualFold(mediaType, "multipart/form-data")
}

func parseMultipartImageRequest(body []byte, contentType string) (*multipartImageRequest, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("unable to parse content-type: %v", err)
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("missing multipart boundary")
	}

	req := &multipartImageRequest{
		ImageURLs: make([]string, 0),
	}
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("unable to read multipart part: %v", err)
		}
		fieldName := part.FormName()
		if fieldName == "" {
			_ = part.Close()
			continue
		}
		partContentType := strings.TrimSpace(part.Header.Get("Content-Type"))

		partData, err := io.ReadAll(part)
		_ = part.Close()
		if err != nil {
			return nil, fmt.Errorf("unable to read multipart field %s: %v", fieldName, err)
		}

		value := strings.TrimSpace(string(partData))
		switch fieldName {
		case "model":
			req.Model = value
			continue
		case "prompt":
			req.Prompt = value
			continue
		case "size":
			req.Size = value
			continue
		case "output_format":
			req.OutputFormat = value
			continue
		case "n":
			if value != "" {
				if parsed, err := strconv.Atoi(value); err == nil {
					req.N = parsed
				}
			}
			continue
		}

		if isMultipartImageField(fieldName) {
			if isMultipartImageURLValue(value) {
				req.ImageURLs = append(req.ImageURLs, value)
				continue
			}
			if len(partData) == 0 {
				continue
			}
			imageURL := buildMultipartDataURL(partContentType, partData)
			req.ImageURLs = append(req.ImageURLs, imageURL)
			continue
		}
		if isMultipartMaskField(fieldName) {
			if len(partData) > 0 || value != "" {
				req.HasMask = true
			}
			continue
		}
	}

	return req, nil
}

func isMultipartImageField(fieldName string) bool {
	return fieldName == "image" || fieldName == "image[]" || strings.HasPrefix(fieldName, "image[")
}

func isMultipartMaskField(fieldName string) bool {
	return fieldName == "mask" || fieldName == "mask[]" || strings.HasPrefix(fieldName, "mask[")
}

func isMultipartImageURLValue(value string) bool {
	if value == "" {
		return false
	}
	loweredValue := strings.ToLower(value)
	return strings.HasPrefix(loweredValue, "data:") || strings.HasPrefix(loweredValue, "http://") || strings.HasPrefix(loweredValue, "https://")
}

func buildMultipartDataURL(contentType string, data []byte) string {
	mimeType := strings.TrimSpace(contentType)
	if mimeType == "" || strings.EqualFold(mimeType, "application/octet-stream") {
		mimeType = http.DetectContentType(data)
	}
	mimeType = normalizeMultipartMimeType(mimeType)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	return fmt.Sprintf("data:%s;base64,%s", mimeType, encoded)
}

func normalizeMultipartMimeType(contentType string) string {
	contentType = strings.TrimSpace(contentType)
	if contentType == "" {
		return ""
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil && mediaType != "" {
		return strings.TrimSpace(mediaType)
	}
	if idx := strings.Index(contentType, ";"); idx > 0 {
		return strings.TrimSpace(contentType[:idx])
	}
	return contentType
}
