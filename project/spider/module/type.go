package module

type Type string // 组建类型

const (
	TYPE_DOWNLOADER Type = "downloader"
	TYPE_ANALYZER   Type = "analyzer"
	TYPE_PIPLINE    Type = "pipeline"
)

// 类型与组建字母映射
var legalTypeLetterMap = map[Type]string{
	TYPE_DOWNLOADER: "D",
	TYPE_ANALYZER:   "A",
	TYPE_PIPLINE:    "P",
}

// 组件字母与类型的映射
var legalLetterTypeMap = map[string]Type{
	"D": TYPE_DOWNLOADER,
	"A": TYPE_ANALYZER,
	"P": TYPE_PIPLINE,
}

// 检车类型
func CheckType(p Type, module Module) bool {

	// 判断空值
	if p == "" || module == nil {
		return false
	}

	switch p {
	// 对每种类型尝试进行转换
	case TYPE_DOWNLOADER:
		if _, ok := module.(Downloader); ok {
			return true
		}
	case TYPE_ANALYZER:
		if _, ok := module.(Analyzer); ok {
			return true
		}
	case TYPE_PIPLINE:
		if _, ok := module.(Pipeline); ok {
			return true
		}
	}
	return false
}

// 在映射种查找是否含有该类型
func LegalType(p Type) bool {
	_, ok := legalTypeLetterMap[p]
	return ok
}

// 类型与字母转换
func typeToLetter(p Type) (bool, string) {
	switch p {
	case TYPE_DOWNLOADER:
		return true, "D"
	case TYPE_ANALYZER:
		return true, "A"
	case TYPE_PIPLINE:
		return true, "p"
	}
	return false, ""
}

// 字母与类型转换
func letterToType(letter string) (bool, Type) {
	switch letter {
	case "D":
		return true, TYPE_DOWNLOADER
	case "A":
		return true, TYPE_ANALYZER
	case "P":
		return true, TYPE_PIPLINE
	}
	return false, ""
}

func GetType(mid MID) (ok bool, p Type) {
	parts, err := SplitMid(mid)
	if err != nil {
		return false, ""
	}
	mt, ok := legalLetterTypeMap[parts[0]]
	return ok, mt
}
