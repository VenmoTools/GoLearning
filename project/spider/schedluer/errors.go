package schedluer

import (
	"log"
	"spider/exceptions"
	"spider/module"
	"spider/tools/buffer"
)

func genError(msg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERRPR_TYPE_SCHEDULER, msg)
}

func genErrorByError(err error) error {
	return exceptions.NewCrawlerErrors(exceptions.ERRPR_TYPE_SCHEDULER, err.Error())
}

func genParameter(msg string) error {
	return exceptions.NewCrawlerErrors(exceptions.ERRPR_TYPE_SCHEDULER, exceptions.NewIllegalParameterError(msg).Error())
}

func sendError(err error, mid module.MID, eb buffer.Pool) (ok bool) {

	var cralwErr exceptions.CrawlerError

	cralwErr, ok = err.(exceptions.CrawlerError)
	if !ok {
		var modType module.Type
		var errType exceptions.ErrorType
		ok, modType = module.GetType(mid)
		if !ok {
			errType = exceptions.ERRPR_TYPE_SCHEDULER
		} else {
			switch modType {
			case module.TYPE_DOWNLOADER:
				errType = exceptions.ERROR_TYPE_DOWNLOADER
			case module.TYPE_ANALYZER:
				errType = exceptions.ERROR_TYPE_ANALYZER
			case module.TYPE_PIPLINE:
				errType = exceptions.ERROR_TYPE_PIPLINE
			}
		}
		cralwErr = exceptions.NewCrawlerErrors(errType, err.Error())
	}
	if eb.Closed() {
		return false
	}

	go func(errs exceptions.CrawlerError) {
		if err := eb.Put(errs); err != nil {
			log.Fatalln("Error Buffer --> The error buffer pool was closed. Ignore error sending.")
		}
		return
	}(cralwErr)
	return true
}
