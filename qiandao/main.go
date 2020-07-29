package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"./goticker"
	"./uuid"
)

//CheckPoint 签到配置实体类
type CheckPoint struct {
	UUID     string `json:"uuid"`     //唯一凭证
	Website  string `json:"website"`  //网站名称
	Link     string `json:"link"`     //签到链接
	Method   string `json:"method"`   //请求方式[GET,POST,PUT]
	Day      int8   `json:"day"`      //[0(每天都签到),1-28]
	Time     string `json:"time"`     //签到时间
	Note     string `json:"note"`     //备注
	Cookie   string `json:"cookie"`   //cookie
	Protocol string `json:"protocol"` //请求协议[raw,form-data,x-www-form-urlencoded]
	Data     string `json:"data"`     //请求参数[请以map方式存放]
	IsTrue   int8   `json:"isTrue"`   //是否为真[1(是),0(否)]
}

//QCheckPoint 当日签到配置实体类
type QCheckPoint struct {
	CheckPoint
	IsRequest int8 `json:"isRequest"` //是否签到过[1(是),0(否)]
}

//ResponseData 返回实体类
type ResponseData struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

func makePath(path string, filename string, iscreate bool, segmentpath ...string) string {
	var paths = path
	if segmentpath != nil {
		for _, s := range segmentpath {
			paths = filepath.Join(paths, s)
		}
	}
	if iscreate {
		os.MkdirAll(paths, os.ModePerm)
	}
	if len(filename) != 0 {
		paths = filepath.Join(paths, filename)
	}
	return paths
}

func copyFile(from string, to string) bool {
	input, err := ioutil.ReadFile(from)
	if err != nil {
		fmt.Println(err)
		return false
	}

	err = ioutil.WriteFile(to, input, os.ModePerm)
	if err != nil {
		return false
	}
	return true
}

func makeFile(file string) {
	f, _ := os.OpenFile(file, os.O_WRONLY|os.O_CREATE, os.ModePerm)
	defer f.Close()
}

func readConf(conf string) string {
	// str, _ := os.Getwd()
	file, err := os.Open(conf)
	if err != nil {
		return ""
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	return string(content)
}

func writeConf(conf string, content string) {
	cont := []byte(content)
	err := ioutil.WriteFile(conf, cont, os.ModePerm)
	if err != nil {
		return
	}
}

//Router 路由器
type Router struct{}

func (router *Router) checkPoints(w http.ResponseWriter, r *http.Request) {

	if r.Method != "GET" {
		http.Error(w, "不允许使用该方法!", http.StatusMethodNotAllowed)
		return
	}
	str, _ := os.Getwd()

	now := time.Now()

	//读取当天配置文件
	path := makePath(str, "qiandao.json."+strconv.Itoa(now.Day()), true, "qiandao", now.Month().String())
	dqcontent := readConf(path)
	if len(dqcontent) < 5 {
		if !copyFile(makePath(str, "qiandao.json", false, "conf"), path) {
			makeFile(path)
		}
		dqcontent = "[]"
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", dqcontent)
	runtime.GC()
}

func (router *Router) addCheckPoint(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "不允许使用该方法!", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-TyPe") != "application/json" {
		http.Error(w, "请设置编码为application/json", http.StatusBadRequest)
		return
	}
	//流式解码器
	br := bufio.NewReader(r.Body)

	checkPoints := new(CheckPoint)

	dec := json.NewDecoder(br)

	if err := dec.Decode(&checkPoints); err != nil {
		http.Error(w, "json object decode error", http.StatusBadRequest)
	} else {
		//初始化
		str, _ := os.Getwd()
		now := time.Now()
		var checkPoint []CheckPoint
		var qCheckPoint []QCheckPoint
		var qCheckPoints QCheckPoint
		dqpath := makePath(str, "qiandao.json."+strconv.Itoa(now.Day()), true, "qiandao", now.Month().String())
		srpath := makePath(str, "qiandao.json", true, "conf")

		//读取当天配置文件 和 源配置文件
		dqcontent := readConf(dqpath)
		srcontent := readConf(srpath)

		//添加数据
		if len(dqcontent) > 5 {
			dqerr := json.Unmarshal([]byte(dqcontent), &qCheckPoint)
			if dqerr != nil {
				http.Error(w, "dq: read conf error", http.StatusBadRequest)
				return
			}
		} else {
			if !copyFile(makePath(str, "qiandao.json", false, "conf"), dqpath) {
				makeFile(dqpath)
			}
		}
		if len(srcontent) > 5 {
			srerr := json.Unmarshal([]byte(srcontent), &checkPoint)
			if srerr != nil {
				http.Error(w, "sr: write conf error", http.StatusBadRequest)
				return
			}
		}

		u, err := uuid.NewV4()
		if err != nil {
			http.Error(w, "generate uuid error", http.StatusBadRequest)
			return
		}

		checkPoints.UUID = u.String()

		checkPoint = append(checkPoint, *checkPoints)

		qCheckPoints.CheckPoint = *checkPoints
		qCheckPoints.IsRequest = 0
		qCheckPoint = append(qCheckPoint, qCheckPoints)

		dqstr, dqerr := json.Marshal(qCheckPoint)
		srstr, srerr := json.Marshal(checkPoint)
		if dqerr != nil {
			http.Error(w, "dq: format error", http.StatusBadRequest)
			return
		}
		if srerr != nil {
			http.Error(w, "sr: format error", http.StatusBadRequest)
			return
		}

		//写入文件
		writeConf(dqpath, string(dqstr))
		writeConf(srpath, string(srstr))

	}
	response := ResponseData{"0000", "ok"}
	jsonData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("JSON marshaling  failed: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", jsonData)
	runtime.GC()
}

func (router *Router) updateCheckPoint(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "不允许使用该方法!", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-TyPe") != "application/json" {
		http.Error(w, "请设置编码为application/json", http.StatusBadRequest)
		return
	}
	//流式解码器
	br := bufio.NewReader(r.Body)

	var checkPoints CheckPoint

	dec := json.NewDecoder(br)

	if err := dec.Decode(&checkPoints); err != nil {
		http.Error(w, "json object decode error", http.StatusBadRequest)
	} else {
		//初始化
		str, _ := os.Getwd()
		now := time.Now()
		var checkPoint []CheckPoint
		var qCheckPoint []QCheckPoint
		dqpath := makePath(str, "qiandao.json."+strconv.Itoa(now.Day()), true, "qiandao", now.Month().String())
		srpath := makePath(str, "qiandao.json", false, "conf")

		//读取当天配置文件 和 源配置文件
		dqcontent := readConf(dqpath)
		srcontent := readConf(srpath)

		//修改当前UUID一致的数据
		if len(dqcontent) > 5 {
			dqerr := json.Unmarshal([]byte(dqcontent), &qCheckPoint)
			if dqerr != nil {
				http.Error(w, "dq: read conf error", http.StatusBadRequest)
				return
			}
		} else {
			if !copyFile(makePath(str, "qiandao.json", false, "conf"), dqpath) {
				makeFile(dqpath)
			}
		}
		if len(srcontent) > 5 {
			srerr := json.Unmarshal([]byte(srcontent), &checkPoint)
			if srerr != nil {
				http.Error(w, "sr: write conf error", http.StatusBadRequest)
				return
			}
		}
		var cPoint []CheckPoint
		var qCPoint []QCheckPoint
		for _, cp := range checkPoint {
			if cp.UUID != checkPoints.UUID {
				cPoint = append(cPoint, cp)
			} else {
				cPoint = append(cPoint, checkPoints)
			}
		}
		for _, qcp := range qCheckPoint {
			if qcp.UUID != checkPoints.UUID {
				qCPoint = append(qCPoint, qcp)
			} else {
				var tmpQCPint QCheckPoint
				tmpQCPint.CheckPoint = checkPoints
				tmpQCPint.IsRequest = qcp.IsRequest
				qCPoint = append(qCPoint, tmpQCPint)
			}
		}

		dqstr, dqerr := json.Marshal(qCPoint)
		srstr, srerr := json.Marshal(cPoint)
		if dqerr != nil {
			http.Error(w, "dq: format error", http.StatusBadRequest)
			return
		}
		if srerr != nil {
			http.Error(w, "sr: format error", http.StatusBadRequest)
			return
		}

		//写入文件
		writeConf(dqpath, string(dqstr))
		writeConf(srpath, string(srstr))
	}

	response := ResponseData{"0000", "ok"}
	jsonData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("JSON marshaling  failed: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", jsonData)
	runtime.GC()
}

func (router *Router) deleteCheckPoint(w http.ResponseWriter, r *http.Request) {

	if r.Method != "POST" {
		http.Error(w, "不允许使用该方法!", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-TyPe") != "application/json" {
		http.Error(w, "请设置编码为application/json", http.StatusBadRequest)
		return
	}

	br := bufio.NewReader(r.Body)

	var checkPoints CheckPoint

	dec := json.NewDecoder(br)

	if err := dec.Decode(&checkPoints); err != nil {
		http.Error(w, "json object decode error", http.StatusBadRequest)
	} else {

		//初始化
		str, _ := os.Getwd()
		now := time.Now()
		var checkPoint []CheckPoint
		var qCheckPoint []QCheckPoint
		dqpath := makePath(str, "qiandao.json."+strconv.Itoa(now.Day()), true, "qiandao", now.Month().String())
		srpath := makePath(str, "qiandao.json", false, "conf")

		//读取当天配置文件 和 源配置文件
		dqcontent := readConf(dqpath)
		srcontent := readConf(srpath)

		//删除当前UUID一致的数据
		if len(dqcontent) > 5 {
			dqerr := json.Unmarshal([]byte(dqcontent), &qCheckPoint)
			if dqerr != nil {
				http.Error(w, "dq: read conf error", http.StatusBadRequest)
				return
			}
		} else {
			if !copyFile(makePath(str, "qiandao.json", false, "conf"), dqpath) {
				makeFile(dqpath)
			}
		}
		if len(srcontent) > 5 {
			srerr := json.Unmarshal([]byte(srcontent), &checkPoint)
			if srerr != nil {
				http.Error(w, "sr: write conf error", http.StatusBadRequest)
				return
			}
		}
		var cPoint []CheckPoint
		var qCPoint []QCheckPoint
		for _, cp := range checkPoint {
			if cp.UUID != checkPoints.UUID {
				cPoint = append(cPoint, cp)
			}
		}
		for _, qcp := range qCheckPoint {
			if qcp.UUID != checkPoints.UUID {
				qCPoint = append(qCPoint, qcp)
			}
		}

		dqstr, dqerr := json.Marshal(qCPoint)
		srstr, srerr := json.Marshal(cPoint)
		if dqerr != nil {
			http.Error(w, "dq: format error", http.StatusBadRequest)
			return
		}
		if srerr != nil {
			http.Error(w, "sr: format error", http.StatusBadRequest)
			return
		}

		//写入文件
		writeConf(dqpath, string(dqstr))
		writeConf(srpath, string(srstr))
	}

	response := ResponseData{"0000", "ok"}
	jsonData, err := json.Marshal(response)
	if err != nil {
		fmt.Printf("JSON marshaling  failed: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, "%s\n", jsonData)
	runtime.GC()
}

//实现http.Handler这个接口的唯一方法
func (router *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	switch urlPath {
	case "/qiandao/checkPoints":
		router.checkPoints(w, r)
	case "/qiandao/updateCheckPoint":
		router.updateCheckPoint(w, r)
	case "/qiandao/addCheckPoint":
		router.addCheckPoint(w, r)
	case "/qiandao/deleteCheckPoint":
		router.deleteCheckPoint(w, r)
	default:
		http.Error(w, "没有此url路径", http.StatusBadRequest)
	}
}

func main() {
	//初始化配置文件
	str, _ := os.Getwd()
	file := makePath(str, "qiandao.json", true, "conf")
	makeFile(file)
	//实例化路由器Handler
	Router := &Router{}
	//基于TCP服务监听8088端口
	ln, err := net.Listen("tcp", ":8088")
	if err != nil {
		fmt.Printf("设置监听端口出错...")
		return
	}
	task := goticker.New(500, true)
	task.AddTaskCallBackFunc(QianDaoTask, 60*30, "QianDaoTask")
	//调用http.Serve(l net.Listener, handler Handler)方法，启动监听
	err1 := http.Serve(ln, Router)
	if err1 != nil {
		fmt.Printf("启动监听出错")
	}

}

//QianDaoTask 任务
func QianDaoTask(args interface{}) {
	//判断文件是否存在
	//读取当天配置文件
	str, _ := os.Getwd()
	now := time.Now()
	path := makePath(str, "qiandao.json."+strconv.Itoa(now.Day()), true, "qiandao", now.Month().String())
	dqcontent := readConf(path)
	if len(dqcontent) < 5 {
		if !copyFile(makePath(str, "qiandao.json", false, "conf"), path) {
			makeFile(path)
		} else {
			dqcontent = readConf(path)
		}
	}
	//读取任务
	var qCheckPoint []QCheckPoint
	if len(dqcontent) > 5 {
		dqerr := json.Unmarshal([]byte(dqcontent), &qCheckPoint)
		if dqerr != nil {
			fmt.Println("read task error")
			return
		}
	}
	//筛选到点未签到任务
	var tmpQCP []QCheckPoint
	for _, qcp := range qCheckPoint {
		if qcp.IsTrue != 0 && qcp.IsRequest != 1 && CheckTime(qcp.Day, qcp.Time) {

			fmt.Println("正在签到:" + qcp.Website + ",签到时间:" + qcp.Time + ",当前签到时间:" + now.Format("01-02 15:04"))
			//判断请求方式并请求
			if qcp.Method == "GET" {
				Get(qcp.Link, qcp.Cookie)
			} else if qcp.Method == "PUT" {
				Put(qcp.Link, qcp.Cookie, qcp.Protocol, qcp.Data)
			} else if qcp.Method == "POST" {
				Post(qcp.Link, qcp.Cookie, qcp.Protocol, qcp.Data)
			} else {
				fmt.Println("暂未支持此种请求方式")
			}
			qcp.IsRequest = 1
		}
		tmpQCP = append(tmpQCP, qcp)
	}

	//回写配置
	dqstr, dqerr := json.Marshal(tmpQCP)
	if dqerr != nil {
		fmt.Println("task: format error")
		return
	}

	//写入文件
	writeConf(path, string(dqstr))
}

//CheckTime 检测签到时间
func CheckTime(day int8, date string) bool {

	now := time.Now()

	nowMinute := now.Hour()*60 + now.Minute()

	d := strings.Split(date, ":")
	hour, _ := strconv.Atoi(d[0])
	minute, _ := strconv.Atoi(d[1])

	qdMinute := hour*60 + minute

	if day != 0 && now.Day() == int(day) && nowMinute > qdMinute {
		return true
	} else if day == 0 && nowMinute > qdMinute {
		return true
	}

	return false
}

//Get Get
func Get(link string, cookie string) {
	//new request
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Cookie", cookie)
	http.DefaultClient.Do(req)
}

//Put Put
func Put(link string, cookie string, protocol string, data1 string) {
	kvs := strings.Split(data1, ";")

	m := make(map[string]string)
	var r http.Request
	r.ParseForm()
	for _, s := range kvs {
		kv := strings.Split(s, "=")
		r.Form.Add(kv[0], kv[1])
		m[kv[0]] = kv[1]
	}

	jsonstr, _ := json.Marshal(m)

	bodystr := strings.TrimSpace(r.Form.Encode())

	var req *http.Request
	var err error
	if protocol == "application/json" {
		req, err = http.NewRequest(http.MethodPut, link, bytes.NewBuffer([]byte(jsonstr)))
	} else {
		req, err = http.NewRequest(http.MethodPut, link, strings.NewReader(bodystr))
	}

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("cookie", cookie)
	req.Header.Set("content-type", protocol)

	http.DefaultClient.Do(req)
}

//Post Post
func Post(link string, cookie string, protocol string, data1 string) {
	kvs := strings.Split(data1, ";")

	m := make(map[string]string)
	var r http.Request
	r.ParseForm()
	for _, s := range kvs {
		kv := strings.Split(s, "=")
		r.Form.Add(kv[0], kv[1])
		m[kv[0]] = kv[1]
	}

	jsonstr, _ := json.Marshal(m)

	bodystr := strings.TrimSpace(r.Form.Encode())

	var req *http.Request
	var err error
	if protocol == "application/json" {
		req, err = http.NewRequest(http.MethodPost, link, bytes.NewBuffer([]byte(jsonstr)))
	} else {
		req, err = http.NewRequest(http.MethodPost, link, strings.NewReader(bodystr))
	}

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Set("cookie", cookie)
	req.Header.Set("content-type", protocol)

	http.DefaultClient.Do(req)
}
