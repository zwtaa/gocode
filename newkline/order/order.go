package main

import (
	"bufio"
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

var rpcFunc = rpc.APIFunc{
	URL: "http://172.31.36.242:28048",
	//URL: "https://api.bituan.io",
}
var wg sync.WaitGroup

func main() {
	start := time.Now()
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
	if v["open"] == "off" {
		return
	}

	for index := 0; index < 6; index++ {
		wg.Add(1)
		go creat("trade", v, k)
		time.Sleep(time.Second * 9)
	}
	cost := time.Since(start)
	wg.Wait()
	fmt.Println(cost)
}

//下单
func creat(types string, v map[string]interface{}, k string) {
	defer wg.Done()
	//desc, _ := strconv.ParseFloat(v["desc"].(string), 64)
	am, _ := strconv.ParseFloat(v["am"].(string), 64)
	userID, _ := strconv.ParseInt(v["depthuser"].(string), 10, 64)
	tradeMin, _ := strconv.ParseFloat(v["tsmall"].(string), 64)
	tradeMax, _ := strconv.ParseFloat(v["tmore"].(string), 64)
	tradeMin *= math.Floor(math.Pow(10, am) + 0.5)
	tradeMax *= math.Floor(math.Pow(10, am) + 0.5)
	tradeNum, err := randFloat64(tradeMin, tradeMax)
	if err != nil {
		return
	}
	tradeNum /= math.Pow(10, am)
	switch types {
	case "trade":
		var price float64
		fmt.Println(v["open"].(string))
		if v["open"] == "off" {
			break
		}
		if v["ot"] == "on" { //买一卖一成交
			res, err := rpcFunc.GetExchangeData("get", v["exchangetrickurl"].(string)+v["nums"].(string))
			if err != nil {
				fmt.Println("exchangetrickurl dep", v)
				break
			}
			fmt.Println(string(res))
			//数据格式化
			reu, err := rpcFunc.StringToData(res, "trick", v["exchangename"].(string))
			if err != nil {
				fmt.Println("StringToData dep", err)
				break
			}
			if v["exchangename"].(string) == "ok" {
				price, _ = strconv.ParseFloat(reu["now"].(string), 10)
			} else {
				price = reu["now"].(float64)
			}
		} else {
			break
		}
		//各种情况生成tradeNUm 和 price
		bsNum := rand.Intn(10)
		var sideArr [2]string
		if bsNum < 5 { //buy
			sideArr[0] = "BUY"
			sideArr[1] = "SELL"
		} else { //sell
			sideArr[0] = "SELL"
			sideArr[1] = "BUY"
		}
		SellRes, err := rpcFunc.CreatOrder(userID, price, sideArr[0], k, tradeNum)
		if err != nil {
			//      ch <- "creat " + sideArr[0] + " order error"
			break
		}
		BuyRes, err := rpcFunc.CreatOrder(userID, price, sideArr[1], k, tradeNum)
		if err != nil {
			//      ch <- SellRes + "creat " + sideArr[1] + " order error"
			break
		}
		//      ch <- sideArr[0] + ":" + SellRes + "|" + sideArr[1] + ":" + BuyRes
		fmt.Println(SellRes, BuyRes)
		break
	default:
		break

	}
}

//获取区间随机数
func randFloat64(min, max float64) (float64, error) {
	if min >= max || min == 0 || max == 0 {
		return max, nil
	}
	fmt.Println(min, max)
	mins, err := strconv.ParseInt((strconv.FormatFloat(min, 'f', -1, 64)), 10, 64)
	fmt.Println(err, mins)
	maxs, err := strconv.ParseInt((strconv.FormatFloat(max, 'f', -1, 64)), 10, 64)
	fmt.Println(err, maxs)
	return strconv.ParseFloat(strconv.FormatInt((rand.Int63n(maxs-mins)+mins), 10), 64)

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
