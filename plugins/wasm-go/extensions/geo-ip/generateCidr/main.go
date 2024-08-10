package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	//"strconv"
	_ "embed"
)

//go:embed ip.merge.txt
var geoipdata string

func CheckCidr(ip, nb string) (string, error) {
	_, ipv4Net, err := net.ParseCIDR(ip + "/" + nb)
	if err != nil {
		log.Println("check cidr failed.", err)
		return "", err
	}
	return ipv4Net.String(), nil
}

func main() {
	//geoipstr := geoip.GetDataString()
	//eg., 1.0.210.128|1.0.211.127

	outFile := "/data/geoCidr.txt"
	f, err := os.Create(outFile)
	if err != nil {
		log.Println("open file failed.", outFile, err)
		return
	}

	defer f.Close()

	geoIpRows := strings.Split(geoipdata, "\n")
	geoIpRows = geoIpRows[:len(geoIpRows)-1]
	for _, row := range geoIpRows {
		//log.Println("geoip segment: ", row)
		tmpArr := strings.Split(row, "|")
		if len(tmpArr) < 7 {
			log.Println("geoIp row field number is less than 7 " + row)
			return
		}

		sip := tmpArr[0]
		eip := tmpArr[1]
		country := tmpArr[2]
		province := tmpArr[4]
		city := tmpArr[5]
		isp := tmpArr[6]

		sipf := net.ParseIP(sip)
		if sipf == nil {
			log.Println("sip is not ip format " + sip)
			return
		}
		sipb := sipf.To4()
		sipall := binary.BigEndian.Uint32(sipb)

		eipf := net.ParseIP(eip)
		if eipf == nil {
			log.Println("eip is not ip format " + eip)
			return
		}
		eipb := eipf.To4()
		eipall := binary.BigEndian.Uint32(eipb)

		netbit := 0
		for i := 31; i >= 0; i-- {
			if (sipall & (1 << i)) != (eipall & (1 << i)) {
				break
			}
			netbit++
		}

		mid := 0
		tmp := 32 - netbit
		for i := 31; i >= tmp; i-- {
			mid |= 1 << i
		}

		netSeg := sipall & uint32(mid)
		netIp0 := netSeg >> 24 & 0xFF
		netIp1 := netSeg >> 16 & 0xFF
		netIp2 := netSeg >> 8 & 0xFF
		netIp3 := netSeg & 0xFF
		netIp := fmt.Sprintf("%d.%d.%d.%d", netIp0, netIp1, netIp2, netIp3)
		cidr := fmt.Sprintf("%s/%d", netIp, netbit)

		log.Printf("cidr:%s sip:%s eip:%s country:%s province:%s city:%s isp:%s", cidr, sip, eip, country, province, city, isp)
		outRow := fmt.Sprintf("%s|%s|%s|%s|%s", cidr, country, province, city, isp)

		_, err := f.WriteString(outRow + "\n")
		if err != nil {
			log.Println("write string failed.", err)
			return
		}

		/*
			res, err := CheckCidr(eip, strconv.FormatUint(uint64(netbit), 10))
			if err != nil {
				log.Println("check cidr failed.",eip, err)
				return
			}

			if cidr != res {
				log.Printf("Error: different cidr. %s  %s  %s  %s", cidr, res, sip, eip)
			} else {
				log.Printf("good cidr. %s  %s  %s  %s", cidr, res, sip, eip)
			}
		*/

	}
}
