package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"newkline/rpc"
	"os"
	"sync"
	"time"
	"wallet/walletsql"
)

var rpcFunc = rpc.APIFunc{
	URL: "http://172.31.36.242:28048",
	//URL: "https://api.bituan.io",
}
var wg sync.WaitGroup
var sqls = walletsql.Sqldata{
	Sqlstr: "mysql",
}

func init() {
	sqls.Db = sqls.Initsql()
}
func main() {
	start := time.Now()
	rand.Seed(time.Now().Unix())
	if len(os.Args) <= 1 {
		return
	}
	k := os.Args[1]
	wg.Add(1)
	go cancel(k)
	wg.Wait()
	fmt.Println(time.Since(start))
}

func cancel(k string) (string, error) {
	defer wg.Done()
	row := sqls.Querys("select listid,id,uid from kline.kline_record where status = ? and coin = ? order by id asc limit 50", 1, k)
	arr := [50]string{}
	i := 0
	str := ""
	var uid int64
	for row.Next() {
		var listid string
		var id string
		var ud int64
		row.Scan(&listid, &id, &ud)
		arr[i] = listid
		uid = ud
		str += "," + id
		i++
	}
	resz, err := rpcFunc.CancelOrderAll(uid, arr[:i])
	if err != nil {
		fmt.Println(err, "canerror1")
		return "", err
	}
	fmt.Println(resz)
	var vz interface{}
	errs := json.Unmarshal([]byte(resz), &vz)
	if errs != nil {
		return resz, nil
	}
	fmt.Println(vz, str)
	if vz.(map[string]interface{})["status"].(float64) == 0 {
		if len(str) > 1 {
			sqls.Querys("delete from kline.kline_record where id in (" + str[1:] + ")")
		}
	}

	return resz, nil
}
