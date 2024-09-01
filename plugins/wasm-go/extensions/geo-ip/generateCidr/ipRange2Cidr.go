package main

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"

	//"strconv"
	_ "embed"
)

//go:embed ip.merge.txt
var geoipdata string

var CIDR2MASK = []uint32{
	0x00000000, 0x80000000, 0xC0000000, 0xE0000000, 0xF0000000, 0xF8000000,
	0xFC000000, 0xFE000000, 0xFF000000, 0xFF800000, 0xFFC00000, 0xFFE00000,
	0xFFF00000, 0xFFF80000, 0xFFFC0000, 0xFFFE0000, 0xFFFF0000, 0xFFFF8000,
	0xFFFFC000, 0xFFFFE000, 0xFFFFF000, 0xFFFFF800, 0xFFFFFC00, 0xFFFFFE00,
	0xFFFFFF00, 0xFFFFFF80, 0xFFFFFFC0, 0xFFFFFFE0, 0xFFFFFFF0, 0xFFFFFFF8,
	0xFFFFFFFC, 0xFFFFFFFE, 0xFFFFFFFF,
}

func main() {
	outFile := "/data/geoCidr.txt"
	f, err := os.Create(outFile)
	if err != nil {
		log.Println("open file failed.", outFile, err)
		return
	}

	defer f.Close()

	geoIpRows := strings.Split(geoipdata, "\n")
	for _, row := range geoIpRows {
		if row == "" {
			log.Println("this row is empty.")
			continue
		}

		log.Println("geoip segment: ", row)
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

		if country == "0" {
			country = ""
		}
		if province == "0" {
			province = ""
		}
		if city == "0" {
			city = ""
		}
		if isp == "0" {
			isp = ""
		}

		if err := parseGeoIpFile(sip, eip, country, province, city, isp, f); err != nil {
			log.Printf("parse geo ip file failed, error:%v", err)
			return
		}
	}
}

func range2cidrList(startIp string, endIp string) []string {
	cidrList := []string{}

	start := uint32(ipToInt(startIp))
	beginStart := start
	end := uint32(ipToInt(endIp))
	for end >= start {
		maxSize := 32
		for maxSize > 0 {
			mask := CIDR2MASK[maxSize-1]
			maskedBase := start & mask

			if maskedBase != start {
				break
			}

			maxSize--
		}

		x := math.Log2(float64(end - start + 1))
		maxDiff := 32 - int(math.Floor(x))
		if maxSize < maxDiff {
			maxSize = maxDiff
		}
		ipStr := intToIP(int(start))
		cidr := fmt.Sprintf("%s/%d", ipStr, maxSize)
		cidrList = append(cidrList, cidr)

		start += uint32(math.Pow(2, float64(32-maxSize)))
		//avoid dead loop for 255.255.255.255
		if start < beginStart {
			break
		}
	}

	return cidrList
}

func parseGeoIpFile(startIp string, endIp string, country string, province string, city string, isp string, f *os.File) error {
	cidrList := range2cidrList(startIp, endIp)
	for _, cidr := range cidrList {
		outRow := fmt.Sprintf("%s|%s|%s|%s|%s", cidr, country, province, city, isp)
		_, err := f.WriteString(outRow + "\n")
		if err != nil {
			log.Println("write string failed.", outRow, err)
			return err
		}
	}

	return nil
}

func ipToInt(ipStr string) int {
	ipSegs := strings.Split(ipStr, ".")
	var ipInt int = 0
	var pos uint = 24
	for _, ipSeg := range ipSegs {
		tempInt, _ := strconv.Atoi(ipSeg)
		tempInt = tempInt << pos
		ipInt = ipInt | tempInt
		pos -= 8
	}
	return ipInt
}

func intToIP(ipInt int) string {
	ipSegs := make([]string, 4)
	var len int = len(ipSegs)
	buffer := bytes.NewBufferString("")
	for i := 0; i < len; i++ {
		tempInt := ipInt & 0xFF
		ipSegs[len-i-1] = strconv.Itoa(tempInt)
		ipInt = ipInt >> 8
	}
	for i := 0; i < len; i++ {
		buffer.WriteString(ipSegs[i])
		if i < len-1 {
			buffer.WriteString(".")
		}
	}
	return buffer.String()
}
