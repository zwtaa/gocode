package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"newkline/rpc"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

var wg sync.WaitGroup
var rpcFunc = rpc.APIFunc{
	URL: "http://172.31.36.242:28048",
}

func main() {
	conf, err := getConfig("../coin.conf")
	if err != nil {
		fmt.Println(err)
	}
	if len(os.Args) <= 1 {
		return
	}
	k := os.Args[1]
	rand.Seed(time.Now().Unix())
	v, ok := conf[k]
	if !ok {
		fmt.Println(k, "not have")
		return
	}
	for index := 0; index < 15; index++ {
		wg.Add(1)
		go active(k, v)
		time.Sleep(time.Second * 4)
	}
	wg.Wait()
}
func active(k string, v map[string]interface{}) {
	defer wg.Done()
	depth, _ := rpcFunc.GetMarket(k)
	var vs interface{}
	err := json.Unmarshal([]byte(depth), &vs)
	if err != nil {
		fmt.Println(err)
		return
	}
	//fmt.Println(vs)
	if vs.(map[string]interface{})["code"].(string) != "0" {
		return
	}
	asks := vs.(map[string]interface{})["data"].(map[string]interface{})["tick"].(map[string]interface{})["asks"].([]interface{})
	bids := vs.(map[string]interface{})["data"].(map[string]interface{})["tick"].(map[string]interface{})["bids"].([]interface{})
	fmt.Println(asks[2].([]interface{})[0].(float64), asks[6].([]interface{})[0].(float64))
	var sellprice [2]float64
	var buyprice [2]float64
	sellprice[0] = getprice(asks[2].([]interface{})[0].(float64), asks[6].([]interface{})[0].(float64), v["desc"].(string))
	sellprice[1] = getprice(asks[1].([]interface{})[0].(float64), asks[5].([]interface{})[0].(float64), v["desc"].(string))
	buyprice[0] = getprice(bids[2].([]interface{})[0].(float64), bids[6].([]interface{})[0].(float64), v["desc"].(string))
	buyprice[1] = getprice(bids[1].([]interface{})[0].(float64), bids[5].([]interface{})[0].(float64), v["desc"].(string))
	fmt.Println(sellprice, buyprice)
	am, _ := strconv.ParseFloat(v["am"].(string), 64)
	id, _ := strconv.ParseInt(v["depthuser"].(string), 10, 64)
	tradeMin, _ := strconv.ParseFloat(v["tsmall"].(string), 64)
	tradeMax, _ := strconv.ParseFloat(v["tmore"].(string), 64)
	tradeMin *= math.Floor(math.Pow(10, am) + 0.5)
	tradeMax *= math.Floor(math.Pow(10, am) + 0.5)
	for index := 0; index < 2; index++ {
		tradeNums, _ := randFloat64(tradeMin, tradeMax)
		tradeNums /= math.Pow(10, am)
		fmt.Println(id, sellprice[index], "sell", k, tradeNums)
		fmt.Println(id, buyprice[index], "buy", k, tradeNums)
		ressell, errs := rpcFunc.CreatOrder(id, sellprice[index], "SELL", k, tradeNums)
		time.Sleep(time.Millisecond * 700)
		resbuy, errb := rpcFunc.CreatOrder(id, buyprice[index], "BUY", k, tradeNums)
		time.Sleep(time.Second * 1)
		fmt.Println(ressell, resbuy)
		arr := [2]string{}
		i := 0
		if errs == nil {
			var vse interface{}
			er := json.Unmarshal([]byte(ressell), &vse)
			if er == nil && vse.(map[string]interface{})["status"].(float64) == 0 {
				arr[i] = vse.(map[string]interface{})["data"].(string)
				i++
			}
		}
		if errb == nil {
			var vsb interface{}
			ed := json.Unmarshal([]byte(resbuy), &vsb)
			if ed == nil && vsb.(map[string]interface{})["status"].(float64) == 0 {
				arr[i] = vsb.(map[string]interface{})["data"].(string)
			}
		}
		if len(arr) > 0 {
			fmt.Println(arr)
			reub, _ := rpcFunc.CancelOrderAll(id, arr[:])
			fmt.Println(reub)
		}

	}
}
func randFloat64(min, max float64) (float64, error) {
	if min >= max || min == 0 || max == 0 {
		return max, nil
	}
	fmt.Println(min, max)
	min, _ = strconv.ParseFloat(fmt.Sprintf("%0.0f", min), 64)
	max, _ = strconv.ParseFloat(fmt.Sprintf("%0.0f", max), 64)
	mins, err := strconv.ParseInt((strconv.FormatFloat(min, 'f', -1, 64)), 10, 64)
	fmt.Println(err, mins)
	maxs, err := strconv.ParseInt((strconv.FormatFloat(max, 'f', -1, 64)), 10, 64)
	fmt.Println(err, maxs)
	return strconv.ParseFloat(strconv.FormatInt((rand.Int63n(maxs-mins)+mins), 10), 64)

}

//获取买卖价格
func getprice(min, max float64, dec string) float64 {
	desc, _ := strconv.ParseFloat(dec, 64)
	min = min*math.Pow(10, desc) + 1
	max = max * math.Pow(10, desc)
	reu, _ := randFloat64(min, max)
	return reu / math.Pow(10, desc)
}

// 获取配置文件
func getConfig(fileName string) (map[string]map[string]interface{}, error) {
	m := make(map[string]map[string]interface{})
	file, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	var lineNo int
	var keys string
	reader := bufio.NewReader(file)
	for {
		line, errRet := reader.ReadString('\n')
		if errRet == io.EOF {
			break
		}
		if errRet != nil {
			return nil, errRet
		}
		lineNo++
		//取出空格
		line = strings.TrimSpace(line)
		if len(line) == 0 || line[0] == '\n' || line[0] == '#' {
			//换行或者#号开头（注释），不计入配置选项
			continue
		}
		arr := strings.Split(line, "=")
		if len(arr) == 0 {
			fmt.Printf("配置文件第%d行存在错误", lineNo)
			continue
		}
		key := strings.TrimSpace(arr[0])
		if len(key) == 0 {
			fmt.Printf("配置文件第%d行存在错误", lineNo)
			continue
		}
		if len(arr[1]) == 0 {
			fmt.Printf("配置文件第%d行存在错误", lineNo)
			continue
		}
		var value string
		if len(arr) > 2 {
			for index := 1; index < len(arr); index++ {
				value = value + "=" + strings.TrimSpace(arr[index])
			}
			value = value[1:]
		} else {
			value = strings.TrimSpace(arr[1])
		}

		if key == "coin" {
			keys = value
		} else {
			if m[keys] == nil {
				m[keys] = make(map[string]interface{})
			}
			m[keys][key] = value
		}
	}
	return m, nil
}
