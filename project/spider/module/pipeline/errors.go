package pipeline

import "spider/exceptions"

func genError(errMsg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERROR_TYPE_PIPLINE, errMsg)
}

// genParameterError 用于生成爬虫参数错误值。
func genParameterError(errMsg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERROR_TYPE_PIPLINE,
		exceptions.NewIllegalParameterError(errMsg).Error())
}
