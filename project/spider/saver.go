package spider

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"spider/datastructs"
	"spider/module"
)



func GenItemProcessors(dirPath string) []module.ProcessItem {
	savePicture := func(item datastructs.Item) (result datastructs.Item, err error) {
		if item == nil {
			return nil, errors.New("invalid item!")
		}

		fmt.Println(item["name"])

		// 检查和准备数据。
		var absDirPath string
		if absDirPath, err = checkDirPath(dirPath); err != nil {
			return
		}
		v := item["reader"]
		reader, ok := v.(io.Reader)
		if !ok {
			return nil, fmt.Errorf("incorrect reader type: %T", v)
		}
		readCloser, ok := reader.(io.ReadCloser)
		if ok {
			defer func() {
				err = readCloser.Close()
			}()
		}
		v = item["name"]
		name, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("incorrect name type: %T", v)
		}
		// 创建图片文件。
		fileName := name
		filePath := filepath.Join(absDirPath, fileName)
		file, err := os.Create(filePath)
		if err != nil {
			return nil, fmt.Errorf("couldn't create file: %s (path: %s)",
				err, filePath)
		}
		defer func() {
			err := file.Close()
			if err != nil {
				fmt.Println(err)
			}
		}()
		// 写图片文件。
		_, err = io.Copy(file, reader)
		if err != nil {
			return nil, err
		}
		// 生成新的条目。
		result = make(map[string]interface{})
		for k, v := range item {
			result[k] = v
		}
		result["file_path"] = filePath
		fileInfo, err := file.Stat()
		if err != nil {
			return nil, err
		}
		result["file_size"] = fileInfo.Size()
		return result, nil
	}


	recordPicture := func(item datastructs.Item) (result datastructs.Item, err error) {
		v := item["file_path"]
		path, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("incorrect file path type: %T", v)
		}
		v = item["file_size"]
		size, ok := v.(int64)
		if !ok {
			return nil, fmt.Errorf("incorrect file name type: %T", v)
		}
		log.Printf("Saved file: %s, size: %d byte(s).", path, size)
		return nil, nil
	}
	return []module.ProcessItem{savePicture, recordPicture}
}

func checkDirPath(dirPath string) (absDirPath string, err error) {
	if dirPath == "" {
		err = fmt.Errorf("invalid dir path: %s", dirPath)
		return
	}
	if filepath.IsAbs(dirPath) {
		absDirPath = dirPath
	} else {
		absDirPath, err = filepath.Abs(dirPath)
		if err != nil {
			return
		}
	}
	var dir *os.File
	dir, err = os.Open(absDirPath)
	if err != nil && !os.IsNotExist(err) {
		return
	}
	if dir == nil {
		err = os.MkdirAll(absDirPath, 0700)
		if err != nil && !os.IsExist(err) {
			return
		}
	} else {
		var fileInfo os.FileInfo
		fileInfo, err = dir.Stat()
		if err != nil {
			return
		}
		if !fileInfo.IsDir() {
			err = fmt.Errorf("not directory: %s", absDirPath)
			return
		}
	}
	return
}