package testIpSegmentToCidr

import (
	"bytes"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"

	//"strconv"
	_ "embed"
)

var CIDR2MASK = []uint32{
	0x00000000, 0x80000000, 0xC0000000, 0xE0000000, 0xF0000000, 0xF8000000,
	0xFC000000, 0xFE000000, 0xFF000000, 0xFF800000, 0xFFC00000, 0xFFE00000,
	0xFFF00000, 0xFFF80000, 0xFFFC0000, 0xFFFE0000, 0xFFFF0000, 0xFFFF8000,
	0xFFFFC000, 0xFFFFE000, 0xFFFFF000, 0xFFFFF800, 0xFFFFFC00, 0xFFFFFE00,
	0xFFFFFF00, 0xFFFFFF80, 0xFFFFFFC0, 0xFFFFFFE0, 0xFFFFFFF0, 0xFFFFFFF8,
	0xFFFFFFFC, 0xFFFFFFFE, 0xFFFFFFFF,
}

func TestRange2CidrList3(t *testing.T) {
	startIp := "1.0.1.0"
	endIp := "2.0.1.112"
	country := "CountryZ"
	province := "ProvinceZ"
	city := "CityZ"
	isp := "ISPZ"

	expectedOutput :=
		`1.0.1.0/24|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.2.0/23|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.4.0/22|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.8.0/21|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.16.0/20|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.32.0/19|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.64.0/18|CountryZ|ProvinceZ|CityZ|ISPZ
1.0.128.0/17|CountryZ|ProvinceZ|CityZ|ISPZ
1.1.0.0/16|CountryZ|ProvinceZ|CityZ|ISPZ
1.2.0.0/15|CountryZ|ProvinceZ|CityZ|ISPZ
1.4.0.0/14|CountryZ|ProvinceZ|CityZ|ISPZ
1.8.0.0/13|CountryZ|ProvinceZ|CityZ|ISPZ
1.16.0.0/12|CountryZ|ProvinceZ|CityZ|ISPZ
1.32.0.0/11|CountryZ|ProvinceZ|CityZ|ISPZ
1.64.0.0/10|CountryZ|ProvinceZ|CityZ|ISPZ
1.128.0.0/9|CountryZ|ProvinceZ|CityZ|ISPZ
2.0.0.0/24|CountryZ|ProvinceZ|CityZ|ISPZ
2.0.1.0/26|CountryZ|ProvinceZ|CityZ|ISPZ
2.0.1.64/27|CountryZ|ProvinceZ|CityZ|ISPZ
2.0.1.96/28|CountryZ|ProvinceZ|CityZ|ISPZ
2.0.1.112/32|CountryZ|ProvinceZ|CityZ|ISPZ
`

	f, err := os.CreateTemp("", "geoCidr_test3.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())
	//defer f.Close()

	range2cidrList(startIp, endIp, country, province, city, isp, f)
	//f.Close()

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != expectedOutput {
		t.Errorf("range2cidrList() output = %s; want %s", string(data), expectedOutput)
	}
}

func TestRange2CidrList2(t *testing.T) {
	startIp := "192.168.0.0"
	endIp := "192.168.0.255"
	country := "CountryX"
	province := "ProvinceX"
	city := "CityX"
	isp := "ISPX"

	expectedOutput := "192.168.0.0/24|CountryX|ProvinceX|CityX|ISPX\n"

	f, err := os.CreateTemp("", "geoCidr_test2.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	range2cidrList(startIp, endIp, country, province, city, isp, f)
	f.Close()

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != expectedOutput {
		t.Errorf("range2cidrList() output = %s; want %s", string(data), expectedOutput)
	}
}

func TestRange2CidrList(t *testing.T) {
	startIp := "224.0.0.0"
	endIp := "255.255.255.255"
	country := "CountryY"
	province := "ProvinceY"
	city := "CityY"
	isp := "ISPY"

	expectedOutput := "224.0.0.0/3|CountryY|ProvinceY|CityY|ISPY\n"

	f, err := os.CreateTemp("", "geoCidr_test.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(f.Name())

	range2cidrList(startIp, endIp, country, province, city, isp, f)
	f.Close()

	data, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatalf("Failed to read temp file: %v", err)
	}

	if string(data) != expectedOutput {
		t.Errorf("range2cidrList() output = %s; want %s", string(data), expectedOutput)
	}
}

func range2cidrList(startIp string, endIp string, country string, province string, city string, isp string, f *os.File) {
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
		outRow := fmt.Sprintf("%s|%s|%s|%s|%s", cidr, country, province, city, isp)
		_, err := f.WriteString(outRow + "\n")
		if err != nil {
			log.Println("write string failed.", outRow, err)
			return
		}

		start += uint32(math.Pow(2, float64(32-maxSize)))
		//avoid dead loop for 255.255.255.255
		if start < beginStart {
			break
		}
	}
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
