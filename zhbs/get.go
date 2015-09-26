package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type Code struct {
	ReCode int    `json:"recode"`
	Msg    string `json:"msg"`
	Result string `json:"result"`
}

type Result struct {
	ReCode int    `json:"recode"`
	Msg    string `json:"msg"`
}

func shakeGet(url string) (r io.Reader, err error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("user-agent", "Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1;SV1)")
	req.Header.Set("connection", "Keep-Alive")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, res.Body)

	return buf, err
}

func step1(phone, id, contentId string) (code *Code, err error) {
	u1 := fmt.Sprintf("http://cms.zhcs0439.com/client/robConfigure/robConfigureCode.do?phoneNumber=%s&id=%s&content_id=%s", phone, id, contentId)
	r, err := shakeGet(u1)
	if err != nil {
		err = fmt.Errorf("get configure code error: %s\n", err)
		return
	}
	//var r io.Reader = bytes.NewReader([]byte(`{"recode":1,"msg":"验证成功，可返回验 证码！","result":"9db17da1314fbdfb5f00ee30b94e89b1"}`))
	code = &Code{}
	decoder := json.NewDecoder(r)
	err = decoder.Decode(code)
	if err != nil {
		err = fmt.Errorf("decode configure code error: %s\n", err)
		return
	}
	if code.ReCode == 0 {
		err = fmt.Errorf("configure code return: %d, not 1, msg: %s\n", code.ReCode, code.Msg)
		return
	}
	return
}

func step2(phone, code, id, contentId string) (result *Result, err error) {
	u2 := fmt.Sprintf("http://cms.zhcs0439.com/client/robConfigure/robConfigure.do?phoneNumber=%s&code=%s&id=%s&content_id=%s", phone, code, id, contentId)
	r, err := shakeGet(u2)
	if err != nil {
		err = fmt.Errorf("get robconfig error: %s\n", err)
		return
	}
	result = &Result{}
	decoder := json.NewDecoder(r)
	err = decoder.Decode(result)
	if err != nil {
		err = fmt.Errorf("decode robconfig error: %s\n", err)
		return
	}

	return
}

func main() {
	// get 13943969701,15843983987 38 3888,3887
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)
	if len(os.Args) < 4 {
		log.Printf("Usage: %s phone id content_id\n", os.Args[0])
		return
	}
	lock := make(chan bool)
	contentIds := strings.Split(os.Args[3], ",")
	phones := strings.Split(os.Args[1], ",")
	for _, phoneStr := range phones {
		go func(phone string) {
			for _, contentId := range contentIds {
				go func(contentId string) {
					for m := 0; m < 500; m++ {
						code, err := step1(phone, os.Args[2], contentId)
						if err != nil {
							log.Printf("[step1:fail:%s:%s:%d] get code error: %s\n", phone, contentId, m, err)
							time.Sleep(time.Millisecond)
							continue
						}
						log.Printf("[step1:info:%s:%d] get code: %s\n", phone, m, code.Result)

						m := 0
						//for m := 0; m < 10; m++ {
						//for n := 0; n < 70; n++ {
						//	go func(i int) {
						result, err := step2(phone, code.Result, os.Args[2], contentId)
						if err != nil {
							log.Printf("[step2:fail:%s:%s:%d] get robconfig error: %s\n", phone, contentId, m, err)
							return
						}

						if result.ReCode == 1 {
							log.Printf("[step2:success:%s:%s:%d] get robconfig return: %d, msg: %s\n", phone, contentId, m, result.ReCode, result.Msg)
							lock <- true
							return
						}
						log.Printf("[step2:fail:%s:%s:%d] get robconfig return: %d, not 1, msg: %s\n", phone, contentId, m, result.ReCode, result.Msg)
						//}(m * n)
						//}
						//time.Sleep(time.Millisecond)
						//}
					}
				}(contentId)
			}
		}(phoneStr)
	}
	<-lock
}
