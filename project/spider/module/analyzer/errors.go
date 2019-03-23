package analyzer

import (
	"spider/exceptions"
)

// genError 用于生成爬虫错误值。
func genError(errMsg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERROR_TYPE_ANALYZER, errMsg)
}

// genParameterError 用于生成爬虫参数错误值。
func genParameterError(errMsg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERROR_TYPE_ANALYZER,
		exceptions.NewIllegalParameterError(errMsg).Error())
}

