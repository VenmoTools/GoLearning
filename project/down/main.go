package main

import (
	"down/core"
	"down/project"
)


func main() {
	req := core.NewGetRequest(project.GetUrl(1), project.ParserCompany)
	eng := core.NewEngine()
	eng.Run(req)
}
