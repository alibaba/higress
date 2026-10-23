package main

import "strings"

func headerSlice2Map(headerSlice [][2]string) map[string][]string {
	headerMap := make(map[string][]string)
	for _, header := range headerSlice {
		k, v := strings.ToLower(header[0]), header[1]
		headerMap[k] = append(headerMap[k], v)
	}
	return headerMap
}

func headerMap2Slice(headerMap map[string][]string) [][2]string {
	headerSlice := make([][2]string, 0, len(headerMap))
	for k, vs := range headerMap {
		for _, v := range vs {
			headerSlice = append(headerSlice, [2]string{k, v})
		}
	}
	return headerSlice
}
