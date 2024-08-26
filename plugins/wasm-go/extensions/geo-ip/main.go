package main

import (
	"errors"
	"net"
	"net/url"
	"strings"

	_ "embed"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
	"github.com/zmap/go-iptree/iptree"
)

//go:embed geoCidr.txt
var geoipdata string

var GeoIpRdxTree *iptree.IPTree
var HaveInitGeoIpDb bool = false

const (
	DefaultRealIpHeader = "X-Forwarded-For"
	OriginSourceType    = "origin-source"
	HeaderSourceType    = "header"
)

func main() {
	wrapper.SetCtx(
		"geo-ip",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type GeoIpConfig struct {
	IpProtocol   string `json:"ip_protocol"`
	IPSourceType string `json:"ip_source_type"`
	IPHeaderName string `json:"ip_header_name"`
}

type GeoIpData struct {
	Cidr     string `json:"cidr"`
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Isp      string `json:"isp"`
}

func parseConfig(json gjson.Result, config *GeoIpConfig, log wrapper.Log) error {
	sourceType := json.Get("ip_source_type")
	if sourceType.Exists() && sourceType.String() != "" {
		switch sourceType.String() {
		case HeaderSourceType:
			config.IPSourceType = HeaderSourceType
		case OriginSourceType:
		default:
			config.IPSourceType = OriginSourceType
		}
	} else {
		config.IPSourceType = OriginSourceType
	}

	header := json.Get("ip_header_name")
	if header.Exists() && header.String() != "" {
		config.IPHeaderName = header.String()
	} else {
		config.IPHeaderName = DefaultRealIpHeader
	}

	ipProtocol := json.Get("ip_protocol")
	if !ipProtocol.Exists() || ipProtocol.String() == "" {
		config.IpProtocol = "ipv4"
	} else {
		config.IpProtocol = strings.ToLower(ipProtocol.String())
	}

	if HaveInitGeoIpDb {
		return nil
	}

	if err := ReadGeoIpDataToRdxtree(log); err != nil {
		log.Errorf("read geoip data failed.%v", err)
		return err
	}

	HaveInitGeoIpDb = true

	return nil
}

func ReadGeoIpDataToRdxtree(log wrapper.Log) error {
	GeoIpRdxTree = iptree.New()

	//eg., cidr country province city isp
	geoIpRows := strings.Split(geoipdata, "\n")
	for _, row := range geoIpRows {
		if row == "" {
			log.Infof("parsed empty line.")
			continue
		}

		pureRow := strings.Trim(row, " ")
		tmpArr := strings.Split(pureRow, "|")
		if len(tmpArr) < 5 {
			return errors.New("geoIp row field number is less than 5 " + row)
		}

		cidr := strings.Trim(tmpArr[0], " ")
		geoIpData := &GeoIpData{
			Cidr:     cidr,
			Country:  strings.Trim(tmpArr[1], " "),
			Province: strings.Trim(tmpArr[2], " "),
			City:     strings.Trim(tmpArr[3], " "),
			Isp:      strings.Trim(tmpArr[4], " "),
		}

		if err := GeoIpRdxTree.AddByString(cidr, geoIpData); err != nil {
			return errors.New("add geoipdata into radix treefailed " + err.Error())
		}

		log.Infof("added geoip data into radixtree: %v", *geoIpData)
	}

	return nil
}

// search geodata using client ip in radixtree.
func SearchGeoIpDataInRdxtree(ip string, log wrapper.Log) (*GeoIpData, error) {
	val, found, err := GeoIpRdxTree.GetByString(ip)
	if err != nil {
		log.Errorf("search geo ip data in raditree failed. %v %s", err, ip)
		return nil, err
	}

	if found {
		return val.(*GeoIpData), nil
	}

	return nil, errors.New("geo ip data not found")
}

func parseIP(source string) string {
	if strings.Contains(source, ".") {
		// parse ipv4
		return strings.Split(source, ":")[0]
	}
	//parse ipv6
	if strings.Contains(source, "]") {
		return strings.Split(source, "]")[0][1:]
	}
	return source
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config GeoIpConfig, log wrapper.Log) types.Action {
	var (
		s   string
		err error
	)
	if config.IPSourceType == HeaderSourceType {
		s, err = proxywasm.GetHttpRequestHeader(config.IPHeaderName)
		if err == nil {
			s = strings.Split(strings.Trim(s, " "), ",")[0]
		}
	} else {
		var bs []byte
		bs, err = proxywasm.GetProperty([]string{"source", "address"})
		s = string(bs)
	}
	if err != nil {
		log.Errorf("get client ip failed. %s %v", config.IPSourceType, err)
		return types.ActionContinue
	}
	clientIp := parseIP(s)
	chkIp := net.ParseIP(clientIp)
	if chkIp == nil {
		log.Errorf("invalid ip[%s].", clientIp)
		return types.ActionContinue
	}

	if config.IpProtocol == "ipv4" && !strings.Contains(clientIp, ".") {
		log.Errorf("client ip is not ipv4 format.%s", clientIp)
		return types.ActionContinue
	}

	//ipv6 will be implemented in the future.
	if config.IpProtocol == "ipv6" || strings.Contains(clientIp, ":") {
		log.Infof("ipv6 will be implemented in the future.%s %s", clientIp, config.IpProtocol)
		return types.ActionContinue
	}

	geoIpData, err := SearchGeoIpDataInRdxtree(clientIp, log)
	if err != nil {
		log.Errorf("search geo info failed.%v", err)
		return types.ActionContinue
	}

	proxywasm.SetProperty([]string{"geo-city"}, []byte(geoIpData.City))
	proxywasm.SetProperty([]string{"geo-province"}, []byte(geoIpData.Province))
	proxywasm.SetProperty([]string{"geo-country"}, []byte(geoIpData.Country))
	proxywasm.SetProperty([]string{"geo-isp"}, []byte(geoIpData.Isp))

	countryEnc := url.QueryEscape(geoIpData.Country)
	provinceEnc := url.QueryEscape(geoIpData.Province)
	cityEnc := url.QueryEscape(geoIpData.City)
	ispEnc := url.QueryEscape(geoIpData.Isp)

	proxywasm.AddHttpRequestHeader("X-Higress-Geo-Country", countryEnc)
	proxywasm.AddHttpRequestHeader("X-Higress-Geo-Province", provinceEnc)
	proxywasm.AddHttpRequestHeader("X-Higress-Geo-City", cityEnc)
	proxywasm.AddHttpRequestHeader("X-Higress-Geo-Isp", ispEnc)

	return types.ActionContinue
}
