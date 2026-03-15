package main

import "fmt"

func print(v any) { fmt.Print(v) }
func println(v any) { fmt.Println(v) }

var app_name string = "Bazic API"

func health() string {
	return "ok"
}

func main() {
	println(app_name)
	var status string = health()
	if (status == "ok") {
		println("service ready")
	} else {
		println("service down")
	}
}

