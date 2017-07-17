package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

func main() {
	data := "{\"action\":2,\"id\":6}"

	body := strings.NewReader(data)
	req, err := http.NewRequest("POST", "http://120.92.234.72:8888/api/login", body)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	fmt.Println("req =", req)

	clinet := &http.Client{}
	resp, err := clinet.Do(req)
	if err != nil {
		fmt.Println("clinet.Do err =", err)
	}

	defer resp.Body.Close()

	ret, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(ret))
}
