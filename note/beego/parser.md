# 注解路由原理

## 变量

```go
var globalRouterTemplate = `package routers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context/param"{{.globalimport}}
)

func init() {
{{.globalinfo}}
}
`
var (
	lastupdateFilename = "lastupdate.tmp" // 临时文件
	commentFilename    string // 
	pkgLastupdate      map[string]int64 //
	genInfoList        map[string][]ControllerComments

	routerHooks = map[string]int{ // 
		"beego.BeforeStatic": BeforeStatic,
		"beego.BeforeRouter": BeforeRouter,
		"beego.BeforeExec":   BeforeExec,
		"beego.AfterExec":    AfterExec,
		"beego.FinishRouter": FinishRouter,
	}

	routerHooksMapping = map[int]string{
		BeforeStatic: "beego.BeforeStatic",
		BeforeRouter: "beego.BeforeRouter",
		BeforeExec:   "beego.BeforeExec",
		AfterExec:    "beego.AfterExec",
		FinishRouter: "beego.FinishRouter",
	}
)

```

## 注解路由流程
1. 根据当前所在包以及app的绝对路径生成commentsRouter_+路径的方式，路径中的斜杠将会替换成_
2. 比较生成的文件名，首先使用getRouterDir获取router路径然后判断该在router路径下生成的文件名是否存在，不存在返回true xxx
3. 调用NewFileSet来生成FileSet对象然后调用parser.ParseDir(返回指定路径的go包名与go文件的映射)
4. 遍历生成的map中的Files，然后在遍历Files中的Decls找到所有类型为 `*ast.FuncDecl`然后获取specDecl.Recv.List[0].Type转换为*ast.StarExpr类型，如果可以转换将调用parserComments
5. 如果f.Doc为空（Controller的注释）则不做处理，否则调用parseComment进行处理
6. 处理@Param内容 [A],处理@Import [B],处理@Filter[C],处理@router[D]
7. 处理完毕后遍历所解析出来的注释，添加Method（处理函数）,Path，使用buildMethodParams，buildFilters，buildImports，创建MethodParams，FilterComments和ImportComments，然后像genInfoList中添加对应的Controller名称和ControllerComments
8. 使用genRouterCode来生成router文件[E]
9. 保存生成的文件并生成时间戳

+ A. 遍历所有提取出来的注释，去除“//”进行字符串的解析，获取内容然后将其添加到params
+ B. 遍历所有提取出来的注释，去除“//”进行字符串的解析，获取内容然后将其添加到imports
+ C. 遍历所有提取出来的注释，去除“//”进行字符串的解析，获取内容然后将其添加到filters，唯一不同的是解析filters有一个filterLoop的标签
+ D. 创建parsedComment实例用来存储params,filters，imports,遍历所有提取出来的注释，去除“//” 根据正则表达式获取path，method，定义如下var routeRegex = regexp.MustCompile(`@router\s+(\S+)(?:\s+\[(\S+)\])?`)
+ E. 添加所有的Controller名称并排序，遍历该Controller所有的ControllerComment根据其内容创建allmethod，params，methodParams，imports，filters，globalimport生成最终的globalinfo(以上操作就是ControllerComment写成代码的形式然后写入创建的文件中)，globalinfo就是生成init函数向GlobalControllerRouter中注册该Controller，然后根据模板替换globalinfo和globalimport字段

```go
pkgRealpath为app的绝对路径，pkgpath的当前包的路径
func parserPkg(pkgRealpath, pkgpath string) error {
    rep := strings.NewReplacer("\\", "_", "/", "_", ".", "_")
    // commentFilename = app的绝对路径+当前包路径
    commentFilename, _ = filepath.Rel(AppPath, pkgRealpath)
    // 生成的文件名为commentsRouter__xx_go_src_TestProject_controllers.go
    commentFilename = commentPrefix + rep.Replace(commentFilename) + ".go"
    // 比较生成的文件之前存在的文件(如果有)
	if !compareFile(pkgRealpath) {
		logs.Info(pkgRealpath + " no changed")
		return nil
	}
	genInfoList = make(map[string][]ControllerComments)
	fileSet := token.NewFileSet()
	astPkgs, err := parser.ParseDir(fileSet, pkgRealpath, func(info os.FileInfo) bool {
		name := info.Name()
		return !info.IsDir() && !strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".go")
	}, parser.ParseComments)

	if err != nil {
		return err
	}
	for _, pkg := range astPkgs {
		for _, fl := range pkg.Files {
			for _, d := range fl.Decls {
				switch specDecl := d.(type) {
				case *ast.FuncDecl:
					if specDecl.Recv != nil {
						exp, ok := specDecl.Recv.List[0].Type.(*ast.StarExpr) // Check that the type is correct first beforing throwing to parser
						if ok {
							parserComments(specDecl, fmt.Sprint(exp.X), pkgpath)
						}
					}
				}
			}
		}
	}
	genRouterCode(pkgRealpath)
	savetoFile(pkgRealpath)
	return nil
}

func parseComment(lines []*ast.Comment) (pcs []*parsedComment, err error) {
	pcs = []*parsedComment{}
	params := map[string]parsedParam{}
	filters := []parsedFilter{}
	imports := []parsedImport{}

	for _, c := range lines {
		t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(t, "@Param") {
			pv := getparams(strings.TrimSpace(strings.TrimLeft(t, "@Param")))
			if len(pv) < 4 {
				logs.Error("Invalid @Param format. Needs at least 4 parameters")
			}
			p := parsedParam{}
			names := strings.SplitN(pv[0], "=>", 2)
			p.name = names[0]
			funcParamName := p.name
			if len(names) > 1 {
				funcParamName = names[1]
			}
			p.location = pv[1]
			p.datatype = pv[2]
			switch len(pv) {
			case 5:
				p.required, _ = strconv.ParseBool(pv[3])
			case 6:
				p.defValue = pv[3]
				p.required, _ = strconv.ParseBool(pv[4])
			}
			params[funcParamName] = p
		}
	}

	for _, c := range lines {
		t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(t, "@Import") {
			iv := getparams(strings.TrimSpace(strings.TrimLeft(t, "@Import")))
			if len(iv) == 0 || len(iv) > 2 {
				logs.Error("Invalid @Import format. Only accepts 1 or 2 parameters")
				continue
			}

			p := parsedImport{}
			p.importPath = iv[0]

			if len(iv) == 2 {
				p.importAlias = iv[1]
			}

			imports = append(imports, p)
		}
	}

filterLoop:
	for _, c := range lines {
		t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(t, "@Filter") {
			fv := getparams(strings.TrimSpace(strings.TrimLeft(t, "@Filter")))
			if len(fv) < 3 {
				logs.Error("Invalid @Filter format. Needs at least 3 parameters")
				continue filterLoop
			}

			p := parsedFilter{}
			p.pattern = fv[0]
			posName := fv[1]
			if pos, exists := routerHooks[posName]; exists {
				p.pos = pos
			} else {
				logs.Error("Invalid @Filter pos: ", posName)
				continue filterLoop
			}

			p.filter = fv[2]
			fvParams := fv[3:]
			for _, fvParam := range fvParams {
				switch fvParam {
				case "true":
					p.params = append(p.params, true)
				case "false":
					p.params = append(p.params, false)
				default:
					logs.Error("Invalid @Filter param: ", fvParam)
					continue filterLoop
				}
			}

			filters = append(filters, p)
		}
	}

	for _, c := range lines {
		var pc = &parsedComment{}
		pc.params = params
		pc.filters = filters
		pc.imports = imports

		t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
		if strings.HasPrefix(t, "@router") {
			t := strings.TrimSpace(strings.TrimLeft(c.Text, "//"))
			matches := routeRegex.FindStringSubmatch(t)
			if len(matches) == 3 {
				pc.routerPath = matches[1]
				methods := matches[2]
				if methods == "" {
					pc.methods = []string{"get"}
					//pc.hasGet = true
				} else {
					pc.methods = strings.Split(methods, ",")
					//pc.hasGet = strings.Contains(methods, "get")
				}
				pcs = append(pcs, pc)
			} else {
				return nil, errors.New("Router information is missing")
			}
		}
	}
	return
}
```
