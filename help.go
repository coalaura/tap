package main

const helpTemplate = `{{.FullName}} - {{.Usage}}

usage: {{.FullName}}{{if .VisibleCommands}} <command>{{end}}{{if .ArgsUsage}} {{.ArgsUsage}}{{end}} [options]
{{if .VisibleCommands}}
commands:{{range .VisibleCommands}}
	{{join .Names ", "}}{{"\t"}}{{.Usage}}{{end}}
{{end}}{{if .VisibleFlags}}
options:{{range .VisibleFlags}}
	{{.}}{{end}}
{{end}}`
