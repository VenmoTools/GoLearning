package module

import (
	"fmt"
	"net"
	"spider/exceptions"
	"strconv"
	"strings"
)

var midTemplate = "%s%d|%s"

type MID string

func SplitMid(mid MID) ([]string, error) {
	var letter string
	var snStr string
	var addr string

	midStr := string(mid)

	if len(mid) <= 1 {
		return nil, exceptions.NewIllegalParameterError("insufficient MID")
	}

	letter = midStr[:1]

	if _, ok := legalLetterTypeMap[letter]; !ok {
		return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module type letter: %s", letter))
	}
	snAddr := midStr[1:]
	index := strings.LastIndex(snAddr, "|")
	if index < 0 {
		snStr = snAddr
		if !legalSn(snStr) {
			return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module SN: %s", snStr))
		}
	} else {
		snStr = snAddr[:index]
		if !legalSn(snStr) {
			return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module SN: %s", snStr))
		}
		addr = snAddr[index+1:]
		index = strings.LastIndex(addr, ":")
		if index <= 0 {
			return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module address: %s", addr))
		}
		ipStr := addr[:index]
		if ip := net.ParseIP(ipStr); ip == nil {
			return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module IP: %s", ipStr))
		}

		portStr := addr[index+1:]
		if _, err := strconv.ParseUint(portStr, 10, 64); err != nil {
			return nil, exceptions.NewIllegalParameterError(fmt.Sprintf("illegal module port: %s", portStr))
		}
	}

	return []string{letter, snStr, addr}, nil
}

func legalSn(str string) bool {
	_, err := strconv.ParseUint(str, 10, 64)
	return err == nil
}

func LegalMid(str MID) bool {
	_, err := SplitMid(str)
	return err == nil
}

// GenMID 会根据给定参数生成组件ID。
func GenMID(mtype Type, sn uint64, maddr net.Addr) (MID, error) {
	if !LegalType(mtype) {
		errMsg := fmt.Sprintf("illegal module type: %s", mtype)
		return "", exceptions.NewIllegalParameterError(errMsg)
	}
	letter := legalTypeLetterMap[mtype]
	var midStr string
	if maddr == nil {
		midStr = fmt.Sprintf(midTemplate, letter, sn, "")
		midStr = midStr[:len(midStr)-1]
	} else {
		midStr = fmt.Sprintf(midTemplate, letter, sn, maddr.String())
	}
	return MID(midStr), nil
}
