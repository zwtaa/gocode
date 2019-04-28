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
	"wallet/walletsql"
)

var rpcFunc = rpc.APIFunc{
	URL: "http://172.31.36.242:28048",
}
var wg sync.WaitGroup
var sql = walletsql.Sqldata{
	Sqlstr: "mysql",
}

func init() {
	sql.Db = sql.Initsql()
}
func main() {
	start := time.Now()
	conf, err := getConfig("./coin.conf")
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
	fmt.Println(v["depthopen"])
	if v["depthopen"] == "off" {
		return
	}
	wg.Add(1)
	go creat("depth", v, k)

	cost := time.Since(start)
	wg.Wait()
	fmt.Println(cost)
}

//下单
func creat(types string, v map[string]interface{}, k string) {
	defer wg.Done()
	dec := v["desc"].(string)
	am, _ := strconv.ParseFloat(v["am"].(string), 64)
	tradeMin, _ := strconv.ParseFloat(v["tsmall"].(string), 64)
	tradeMax, _ := strconv.ParseFloat(v["tmore"].(string), 64)
	tradeMin *= math.Floor(math.Pow(10, am) + 0.5)
	tradeMax *= math.Floor(math.Pow(10, am) + 0.5)
	tradeNum, err := randFloat64(tradeMin, tradeMax)
	if err != nil {
		return
	}
	fmt.Println(tradeNum)
	tradeNum /= math.Pow(10, am)
	switch types {
	case "depth":
		if v["depthopen"] == "off" {
			fmt.Println("depthopen off", v)
			break
		}
		var result float64
		if v["depthexchange"] == "on" { //摆盘拉取交易所
			//拉取最新价格
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
				result, _ = strconv.ParseFloat(reu["now"].(string), 10)
			} else {
				result = reu["now"].(float64)
			}
		} else if v["depthself"] == "on" { //摆盘拉取币种涨跌幅
			//https://open.bituan.cc/v1/get_ticker?symbol=btcusdt
			url := "https://open.bituan.cc/v1/get_ticker?symbol=" + v["depthcoin"].(string) + "usdt"
			res, err := rpcFunc.GetExchangeData("get", url)
			if err != nil {
				fmt.Println("self error", err)
				break
			}
			var vc interface{}
			errs := json.Unmarshal([]byte(res), &vc)
			if errs != nil {
				fmt.Println("self error json", err)
				break
			}
			rose := vc.(map[string]interface{})["data"].(map[string]interface{})["rose"].(float64)
			urls := "https://open.bituan.cc/v1/get_ticker?symbol=" + k
			ress, erra := rpcFunc.GetExchangeData("get", urls)
			if erra != nil {
				fmt.Println("selfa error", erra)
				break
			}
			var vca interface{}
			errsa := json.Unmarshal([]byte(ress), &vca)
			if errsa != nil {
				fmt.Println("selfaa error json", err)
				break
			}
			open := vca.(map[string]interface{})["data"].(map[string]interface{})["open"].(float64)
			//nowpri, _ := strconv.ParseFloat(fmt.Sprintf("%.6f", open*(1+rose)), 64)
			scv := fmt.Sprintf("%.6f", open*(1+rose))
			nowpri, _ := strconv.ParseFloat(scv[:7], 64)
			id, _ := strconv.ParseInt(v["depthuser"].(string), 10, 64)
			wnum, _ := strconv.ParseInt(v["depthnum"].(string), 10, 64)
			buyprice := nowpri
			sellprice := nowpri
			rand.Seed(time.Second.Nanoseconds())
			for index := 0; index < int(wnum); index++ {
				tradeNums, _ := randFloat64(tradeMin, tradeMax)
				tradeNums /= math.Pow(10, am)
				ns := float64(float64(rand.Intn(4)) / math.Pow(10, 6))
				buyprice -= ns
				sellprice += ns
				buyprice, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", buyprice), 64)
				sellprice, _ = strconv.ParseFloat(fmt.Sprintf("%.6f", sellprice), 64)
				buyres, _ := rpcFunc.CreatOrder(id, buyprice, "BUY", k, tradeNums)
				sellres, _ := rpcFunc.CreatOrder(id, sellprice, "SELL", k, tradeNums)
				fmt.Println(buyres, sellres, id, scv, ns)
			}
			break
		}
		id, _ := strconv.ParseInt(v["depthuser"].(string), 10, 64)
		wnum, _ := strconv.ParseInt(v["depthnum"].(string), 10, 64)
		ordernum := int(wnum)
		fmt.Println(id, ordernum)
		buyminnow, buymaxnow, _ := getprice(result, dec, "buy")
		sellminnow, sellmaxnow, _ := getprice(result, dec, "sell")
		row := sql.Querys("select id,types from kline.kline_record where status = ? and coin = ? and ((types = 'buy' and (price < ? or price > ?)) or (types = 'sell' and (price > ? or price < ?)))", 0, k, buyminnow, buymaxnow, sellmaxnow, sellminnow)
		fmt.Println(0, k, buyminnow, buymaxnow, sellmaxnow, sellminnow)
		var str string
		buynum, sellnum := 0, 0
		for row.Next() {
			var ids, types string
			row.Scan(&ids, &types)
			str = str + "," + ids
			if types == "sell" {
				sellnum++
			} else {
				buynum++
			}
			fmt.Println(ids, types)
		}
		if len(str) > 1 {
			fmt.Println(str, sellnum, buynum)
			sql.Querys("update kline.kline_record set status = 1 where id in (" + str[1:] + ")")
		}
		rows := sql.Querys("select count(*) as num,types from kline.kline_record where status = ? and coin = ? group by types ", 0, k)
		realbuynum, realsellnum := 0, 0
		for rows.Next() {
			var realnum int
			var realtype string
			rows.Scan(&realnum, &realtype)
			if realtype == "sell" {
				realsellnum = realnum
			} else {
				realbuynum = realnum
			}
		}
		forsell, forbuy := 0, 0
		//计算补多少卖单
		if realsellnum == 0 {
			forsell = ordernum
		} else if realsellnum >= ordernum*2 {
			forsell = 0
		} else {
			if realsellnum < ordernum {
				forsell = ordernum - realsellnum
			} else {
				forsell = sellnum
			}

		}
		//计算补多少买单
		if realbuynum == 0 {
			forbuy = ordernum
		} else if realbuynum >= ordernum*2 {
			forbuy = 0
		} else {
			if realbuynum < ordernum {
				forbuy = ordernum - realbuynum
			} else {
				forbuy = buynum
			}
		}
		fmt.Println("pppppppppppp", forbuy, forsell)
		//{"status":0,"message":"成功","error":null,"data":"EX201904221327280210000011821","timestamp":"2019-04-22T05:27:28.051+0000","success":true}
		sqlstr := ""
		for index := 0; index < forbuy; index++ {
			tradeNums, _ := randFloat64(tradeMin, tradeMax)
			tradeNums /= math.Pow(10, am)
			_, _, buyprice := getprice(result, dec, "buy")
			fmt.Println(buyprice)
			buyres, _ := rpcFunc.CreatOrder(id, buyprice, "BUY", k, tradeNums)
			//buyres := `{"status":0,"message":"成功","error":null,"data":"EX201904221327280210000011821","timestamp":"2019-04-22T05:27:28.051+0000","success":true}`
			var vz interface{}
			erra := json.Unmarshal([]byte(buyres), &vz)
			if erra != nil {
				continue
			}

			if vz.(map[string]interface{})["status"].(float64) == 0 {
				sqlstr += "," + "('" + k + "','" + vz.(map[string]interface{})["data"].(string) + "','buy'," + fmt.Sprintf("%0."+dec+"f", buyprice) + "," + v["depthuser"].(string) + ")"
			}
		}
		for index := 0; index < forsell; index++ {
			tradeNums, _ := randFloat64(tradeMin, tradeMax)
			tradeNums /= math.Pow(10, am)
			_, _, sellprice := getprice(result, dec, "sell")
			fmt.Println(sellprice)
			sellres, _ := rpcFunc.CreatOrder(id, sellprice, "SELL", k, tradeNums)
			//sellres := `{"status":0,"message":"成功","error":null,"data":"EX201904221327280210000011821","timestamp":"2019-04-22T05:27:28.051+0000","success":true}`
			var vs interface{}
			erra := json.Unmarshal([]byte(sellres), &vs)
			if erra != nil {
				continue
			}
			if vs.(map[string]interface{})["status"].(float64) == 0 {
				sqlstr += "," + "('" + k + "','" + vs.(map[string]interface{})["data"].(string) + "','sell'," + fmt.Sprintf("%0."+dec+"f", sellprice) + "," + v["depthuser"].(string) + ")"
			}
		}
		if len(sqlstr) > 1 {
			sqls := "insert into kline.kline_record (coin,listid,types,price,uid) values " + sqlstr[1:]
			fmt.Println(sqls)
			sql.Querys(sqls)
		}

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
	min, _ = strconv.ParseFloat(fmt.Sprintf("%0.0f", min), 64)
	max, _ = strconv.ParseFloat(fmt.Sprintf("%0.0f", max), 64)
	mins, err := strconv.ParseInt((strconv.FormatFloat(min, 'f', -1, 64)), 10, 64)
	fmt.Println(err, mins)
	maxs, err := strconv.ParseInt((strconv.FormatFloat(max, 'f', -1, 64)), 10, 64)
	fmt.Println(err, maxs)
	return strconv.ParseFloat(strconv.FormatInt((rand.Int63n(maxs-mins)+mins), 10), 64)

}

//获取买卖价格
func getprice(price float64, dec, types string) (float64, float64, float64) {
	desc, _ := strconv.ParseFloat(dec, 64)
	var reu, min, max float64
	switch types {
	case "buy":
		buy, _ := strconv.ParseFloat(fmt.Sprintf("%0."+dec+"f", price*0.999), 64)
		min = buy * math.Pow(10, desc)
		max = price*math.Pow(10, desc) - 1
	case "sell":
		sell, _ := strconv.ParseFloat(fmt.Sprintf("%0."+dec+"f", price*1.001), 64)
		min = price*math.Pow(10, desc) + 1
		max = sell * math.Pow(10, desc)
	default:
		return 0, 0, 0

	}
	reu, _ = randFloat64(min, max)
	return min / math.Pow(10, desc), max / math.Pow(10, desc), reu / math.Pow(10, desc)
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
