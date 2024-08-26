package main

import (
	"errors"
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

func main() {
	wrapper.SetCtx(
		"geo-ip",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type GeoIpConfig struct {
	IpProtocol string `json:"ip_protocol"`
}

type GeoIpData struct {
	Cidr     string `json:"cidr"`
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Isp      string `json:"isp"`
}

func parseConfig(json gjson.Result, config *GeoIpConfig, log wrapper.Log) error {
	ipProtocol := json.Get("ipProtocol")
	if !ipProtocol.Exists() {
		config.IpProtocol = "ipv4"
	} else {
		config.IpProtocol = strings.ToLower(json.Get("ipProtocol").String())
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

func onHttpRequestHeaders(ctx wrapper.HttpContext, config GeoIpConfig, log wrapper.Log) types.Action {
	var clientIp string
	xffHdr, err := proxywasm.GetHttpRequestHeader("x-forwarded-for")
	if err != nil {
		log.Errorf("no request header x-forwarded-for.%v", err)
		remoteAddr, err := proxywasm.GetProperty([]string{"source", "address"})
		if err != nil {
			log.Errorf("get property source address failed.%v", err)
			return types.ActionContinue
		} else {
			clientIp = string(remoteAddr)
			log.Infof("client ip:%s", clientIp)
		}
	} else {
		log.Infof("xff header: %s", xffHdr)
		clientIp = strings.Trim((strings.Split(xffHdr, ","))[0], " ")
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
