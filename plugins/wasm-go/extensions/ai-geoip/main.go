package main

import (
	"ai-geoip/geoipdata"
	"encoding/binary"
	"encoding/json"
	"errors"
	"net"
	"strings"

	"github.com/alibaba/higress/plugins/wasm-go/pkg/wrapper"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm"
	"github.com/higress-group/proxy-wasm-go-sdk/proxywasm/types"
	"github.com/tidwall/gjson"
)



func main() {
	wrapper.SetCtx(
		"ai-geoip",
		wrapper.ParseConfigBy(parseConfig),
		wrapper.ProcessRequestHeadersBy(onHttpRequestHeaders),
	)
}

type AIGeoIpConfig struct {
	IpProtocol string `json:"ip_protocol"`
}

type GeoIpData struct {
	StartIp  uint32 `json:"start_ip"`
	EndIp    uint32 `json:"end_ip"`
	Country  string `json:"country"`
	Province string `json:"province"`
	City     string `json:"city"`
	Isp      string `json:"isp"`
}

type GeoIpVectorItem struct {
	GeoIpDataArr []GeoIpData `json:"geo_ip_data_arr"`
}

var Vector [256][256]GeoIpVectorItem
var HaveInitGeoIpDb bool = false

func parseConfig(json gjson.Result, config *AIGeoIpConfig, log wrapper.Log) error {
	config.IpProtocol = json.Get("ipProtocol").String()

	if HaveInitGeoIpDb {
		return nil
	}

	if err := ReadGeoIpData(log); err != nil {
		log.Errorf("read geoip data failed.%v", err)
		return err
	}

	HaveInitGeoIpDb = true

	return nil
}

func ReadGeoIpData(log wrapper.Log) error {
	//Vector = [256][256]GeoIpVectorItem{}
	geoIpData := geoipdata.GetDataString()
	geoIpRows := strings.Split(geoIpData, "\n")
	for _, row := range geoIpRows {
		log.Errorf("geoip segment: ", row)
		tmpArr := strings.Split(row, "|")
		if len(tmpArr) < 7 {
			return errors.New("geoIp row field number is less than 7 " + row)
		}

		sip := tmpArr[0]
		eip := tmpArr[1]
		country := tmpArr[2]
		province := tmpArr[4]
		city := tmpArr[5]
		isp := tmpArr[6]

		sipf := net.ParseIP(sip)
		if sipf == nil {
			return errors.New("sip is not ip format " + sip)
		}
		sipb := sipf.To4()
		sipall := binary.BigEndian.Uint32(sipb)
		sip0 := (sipall >> 24) & 0xFF
		sip1 := (sipall >> 16) & 0xFF

		eipf := net.ParseIP(eip)
		if eipf == nil {
			return errors.New("eip is not ip format " + eip)
		}
		eipb := eipf.To4()
		eipall := binary.BigEndian.Uint32(eipb)
		eip0 := (eipall >> 24) & 0xFF
		eip1 := (eipall >> 16) & 0xFF

		geoIpData := &GeoIpData{
			StartIp:  sipall,
			EndIp:    eipall,
			Country:  country,
			Province: province,
			City:     city,
			Isp:      isp,
		}

		sgiarr := &(Vector[sip0][sip1].GeoIpDataArr)
		*sgiarr = append(*sgiarr, *geoIpData)

		//for different first 2 segments in start ip and end ip
		if sip0 != eip0 || sip1 != eip1 {
			egiarr := &(Vector[eip0][eip1].GeoIpDataArr)
			*egiarr = append(*egiarr, *geoIpData)
		}
	}
	return nil
}

func BinarySearchIp(ip uint32, arr *[]GeoIpData, log wrapper.Log) (*GeoIpData, error) {
	low := 0
	high := len(*arr) - 1
	for low <= high {
		mid := (high + low) / 2
		startIp := ((*arr)[mid]).StartIp
		endIp := ((*arr)[mid]).EndIp
		log.Errorf("binary search ip. low:%d   high:%d   mid:%d   clientip:%d   startip:%d    endip:%d", low, high, mid, ip, startIp, endIp)
		if ip < startIp {
			high = mid - 1
		} else if ip > endIp {
			low = mid + 1
		} else {
			return &((*arr)[mid]), nil
		}
	}
	return nil, errors.New("ip is not found in geoipdata array")
}

func SearchGeoInfo(ip string, log wrapper.Log) (*GeoIpData, error) {
	ipf := net.ParseIP(ip)
	if ipf == nil {
		log.Errorf("ip is not ip format.%s", ip)
		return nil, errors.New("ip is not ip format")
	}
	ipb := ipf.To4()
	ipall := binary.BigEndian.Uint32(ipb)
	ip0 := ipall >> 24 & 0xFF
	ip1 := ipall >> 16 & 0xFF
	vctItem := Vector[ip0][ip1]
	geoIpDataArr := vctItem.GeoIpDataArr
	if len(geoIpDataArr) == 0 {
		return nil, errors.New("geoip data array empty")
	}
	geoIpData, err := BinarySearchIp(ipall, &geoIpDataArr, log)
	if err != nil {
		log.Errorf("binary search ip failed.%v", err)
		return nil, err
	}

	return geoIpData, nil
}

func onHttpRequestHeaders(ctx wrapper.HttpContext, config AIGeoIpConfig, log wrapper.Log) types.Action {
	var clientIp string
	xffHdr, err := proxywasm.GetHttpRequestHeader("x-forwarded-for")
	if err != nil {
		log.Errorf("no request header x-forwarded-for.%v", err)
		remoteAddr, err := proxywasm.GetProperty([]string{"source","address"})
		if err != nil {
			log.Errorf("get property source address failed.%v", err)
			return types.ActionContinue
		} else {
			clientIp = string(remoteAddr)
			log.Errorf("client ip:%s", clientIp)
		}			
	} else {
		log.Errorf("xff header: ", xffHdr)
		clientIp = strings.Trim((strings.Split(xffHdr, ","))[0], " ")
	}

	if config.IpProtocol == "ipv4" && !strings.Contains(clientIp, ".") {
		log.Errorf("client ip is not ipv4 format.%s", clientIp)
		return types.ActionContinue
	}

	//ipv6 will be implemented in the future.
	if config.IpProtocol == "ipv6" || strings.Contains(clientIp, ":") {
		log.Errorf("ipv6 and will be implemented in the future.%s %s", clientIp, config.IpProtocol)
		return types.ActionContinue
	}

	geoIpData, err := SearchGeoInfo(clientIp, log)
	if err != nil {
		log.Errorf("search geo info failed.%v", err)
		return types.ActionContinue
	}

	geoIpPro, err := json.Marshal(geoIpData)
	if err != nil {
		log.Errorf("marshal geoip data failed.%v", err)
		return types.ActionContinue
	}
	proxywasm.SetProperty([]string{"geoIpData"}, geoIpPro)

	return types.ActionContinue
}
