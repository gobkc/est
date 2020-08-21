package est

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

//Est elasticsearch
type Est struct {
	user         string
	password     string
	host         string
	port         uint
	protocol     string //可选值 http,https
	table        string //索引，这里为了方便称呼，设置为table
	auth         string //通过user和password加密而成
	sort         string //排序方式 asc desc
	page         uint   //当前页 默认为0,传入此参数会自动计算分页
	pageSize     uint   //每一页有多少行数据
	pageNum      uint   //总页数
	addAPI       string //添加数据的restFul API
	saveAPI      string //修改数据的restFul API
	deleteAPI    string //删除数据的restFul API
	findAPI      string //查找数据的restFul API
	id           string //ID，每一行数据的ID，在es里表现为_id
	conditions   []KV   //保存条件，可用于save和find方法
	conditionsOr []KV   //保存OR条件，可用于save和find方法,注意elasticsearch和mysql不一样的地方，MYSQL是最右原则，es是 AND OR分开的，必须要有序
}

//M 用于给elasticsearch添加数据的快捷数据类型
type M map[string]interface{}

//KV 用于保存KEY=>value键值对
type KV struct {
	K string `json:"k"`
	V string `json:"v"`
	C string `json:"c"` //用于存储符号 > < = >= <= <>
}

//NewEst 创建est对象
func NewEst() *Est {
	return &Est{protocol: "http", host: "127.0.0.1", port: 9200, table: "log", page: 0, pageSize: 1000, pageNum: 0}
}

//SetProtocol 设置协议
func (e *Est) SetProtocol(protocol string) *Est {
	if protocol != "http" && protocol != "https" {
		return e
	}
	e.protocol = protocol
	return e
}

//SetPort 设置端口
func (e *Est) SetPort(port uint) *Est {
	e.port = port
	return e
}

//SetHost 设置主机地址
func (e *Est) SetHost(host string) *Est {
	e.host = host
	return e
}

//SetPassword 设置密码
func (e *Est) SetPassword(password string) *Est {
	e.password = password
	return e
}

//SetUser 设置用户
func (e *Est) SetUser(user string) *Est {
	e.user = user
	return e
}

//Table 设置索引（表）
func (e *Est) Table(table string) *Est {
	e.table = table
	return e
}

//SetSort 设置排序字段
func (e *Est) SetSort(filed string, sort string) *Est {
	e.sort = fmt.Sprintf("&sort=%s:%s", filed, sort)
	return e
}

//Where 条件
func (e *Est) Where(condition string, values ...interface{}) *Est {
	//如果条件里面不包含？号，直接退出逻辑，强制要求条件如下: Where("id=?",1)
	if !strings.Contains(condition, "?") {
		return e
	}
	for _, value := range values {
		condition = strings.Replace(condition, "?", fmt.Sprintf("%v", value), 1)
	}
	//condition example : id=1 AND name=xiong OR age=18
	//将条件转换为 key => value struct,注意保留 AND OR
	//将所有的 or 和 and 转换为大写
	rep, _ := regexp.Compile(" or | and ")
	condition = rep.ReplaceAllStringFunc(condition, strings.ToUpper)
	//找出and和or的位置
	andPos := strings.Index(condition, " AND ")
	orPos := strings.Index(condition, " OR ")

	//如果不存在and和or直接传递给条件数组
	if andPos == -1 && orPos == -1 {
		//将符号的两边都加上空格，便于取出
		rep, _ := regexp.Compile(">=|<=|>|<|=|<>")
		condition = rep.ReplaceAllStringFunc(condition, func(s string) string {
			return " " + s + " "
		})

		//获取KV
		kvSplit := strings.Split(strings.TrimSpace(condition), " ")
		if len(kvSplit) < 3 {
			return e
		}
		k := strings.TrimSpace(kvSplit[0])
		v := strings.TrimSpace(kvSplit[2])
		c := strings.TrimSpace(kvSplit[1])
		if k == "id" {
			e.id = v
		} else {
			e.conditions = append(e.conditions, KV{K: k, V: v, C: c})
		}
		return e
	}
	//todo 多条件待做
	return e
}

//clearData 清空数据
func (e *Est) clearData() {
	e.table = ""
	e.id = ""
	e.sort = ""
	e.page = 0
	e.pageSize = 1000
	e.pageNum = 0
}

//Add 添加数据
func (e *Est) Add(data M) (m M, err error) {
	if e.addAPI == "" {
		e.addAPI = fmt.Sprintf("%s://%s:%d/%s/_doc", e.protocol, e.host, e.port, e.table)
	}
	var mJSON []byte
	if mJSON, err = json.Marshal(data); err != nil {
		return m, errors.New("传递的数据非法")
	}
	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()
	return e.do("POST", e.addAPI, mJSON)
}

//Save 添加数据
func (e *Est) Save(data M) (m M, err error) {
	//判断是否有ID，ID参数通过where方法传递
	if e.id == "" {
		return m, errors.New(`ID不存在，请使用Where("id=?",1)来传递`)
	}
	//saveAPI路由中会有变化的ID，所以不能缓存起来，只能用一次生成一次
	e.saveAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/%s/_update?pretty", e.protocol, e.host, e.port, e.table, e.id)
	var mJSON []byte
	if mJSON, err = json.Marshal(M{"doc": data}); err != nil {
		return m, errors.New("传递的数据非法")
	}
	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()
	return e.do("POST", e.saveAPI, mJSON)
}

//SetInc 递增数据 等价于 a = a+1
func (e *Est) SetInc(filedName string, value float64) (m M, err error) {
	//判断是否有ID，ID参数通过where方法传递
	if e.id == "" {
		return m, errors.New(`ID不存在，请使用Where("id=?",1)来传递`)
	}
	//saveAPI路由中会有变化的ID，所以不能缓存起来，只能用一次生成一次
	e.saveAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/%s/_update?pretty", e.protocol, e.host, e.port, e.table, e.id)
	var mJSON []byte
	updateScript := fmt.Sprintf("ctx._source.%s+=%v", filedName, value)
	if mJSON, err = json.Marshal(M{"script": updateScript}); err != nil {
		return m, errors.New("传递的数据非法")
	}
	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()
	return e.do("POST", e.saveAPI, mJSON)
}

//SetDec 递减数据 等价于 a = a-1
func (e *Est) SetDec(filedName string, value float64) (m M, err error) {
	//判断是否有ID，ID参数通过where方法传递
	if e.id == "" {
		return m, errors.New(`ID不存在，请使用Where("id=?",1)来传递`)
	}
	//saveAPI路由中会有变化的ID，所以不能缓存起来，只能用一次生成一次
	e.saveAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/%s/_update?pretty", e.protocol, e.host, e.port, e.table, e.id)
	var mJSON []byte
	updateScript := fmt.Sprintf("ctx._source.%s-=%v", filedName, value)
	if mJSON, err = json.Marshal(M{"script": updateScript}); err != nil {
		return m, errors.New("传递的数据非法")
	}
	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()
	return e.do("POST", e.saveAPI, mJSON)
}

//Delete 删除指定ID的记录  ID通过Where方法传递
func (e *Est) Delete() (m M, err error) {
	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()
	//判断是否有ID，ID参数通过where方法传递
	if e.id == "" {
		return m, errors.New(`ID不存在，请使用Where("id=?",1)来传递`)
	}
	//deleteAPI路由中会有变化的ID，所以不能缓存起来，只能用一次生成一次
	e.deleteAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/%s", e.protocol, e.host, e.port, e.table, e.id)
	var mJSON []byte
	e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	return e.do("DELETE", e.deleteAPI, mJSON)
}

//SetPage 设置当前是多少页，如果设置了此参数，使用find找到的数据里面包含total,page-num,page-size,current-page等选项
func (e *Est) SetPage(page uint) *Est {
	//只要调用了此方法，至少保证有一页
	if page < 1 {
		page = 1
	}
	//为了性能着想，一旦分页，每页的记录数最好是10条(不要大于1000)
	if e.pageSize >= 1000 {
		e.pageSize = 10
	}
	e.page = page
	return e
}

//SetPageSize 设置每页行数
func (e *Est) SetPageSize(pageSize uint) *Est {
	//为了性能着想，一旦分页，每页的记录数最好是10条(不要大于1000)
	if pageSize >= 1000 {
		e.pageSize = 10
	} else {
		e.pageSize = pageSize
	}
	return e
}

//Find 查找数据
func (e *Est) Find() (m interface{}, err error) {
	//判断是否有ID，ID参数通过where方法传递
	if e.table == "" {
		return m, errors.New(`table不存在，请使用SetTable("tableName")`)
	}
	//从哪条记录开始读
	var from uint = 0
	if e.page != 0 {
		from = (e.page - 1) * e.pageSize
	}
	e.findAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/_search?from=%v&size=%v%s",
		e.protocol,
		e.host,
		e.port,
		e.table,
		from,
		e.pageSize,
		e.sort,
	)
	//获取条件
	var tag []string
	for _, condition := range e.conditions {
		switch condition.C {
		case "=":
			//判断值是否可以作为数字
			_, err := strconv.ParseFloat(condition.V, 64)
			newV := condition.V
			if err != nil {
				newV = fmt.Sprintf(`"%v"`, condition.V)
			}
			tag = append(tag, fmt.Sprintf(`{ "match": { "%s": {"query":%s,"minimum_should_match":"100%s"} } }`, condition.K, newV, "%"))
			break
		case ">":
			tag = append(tag, fmt.Sprintf(`{ "range": { "%s": { "gte": %v } } }`, condition.K, condition.V))
			break
		case ">=":
			tag = append(tag, fmt.Sprintf(`{ "range": { "%s": { "gt": %v } } }`, condition.K, condition.V))
			break
		case "<":
			tag = append(tag, fmt.Sprintf(`{ "range": { "%s": { "lte": %v } } }`, condition.K, condition.V))
			break
		case "<=":
			tag = append(tag, fmt.Sprintf(`{ "range": { "%s": { "lt": %v } } }`, condition.K, condition.V))
			break
		}
	}
	var json = fmt.Sprintf(`{ "query": { "bool": { "must": [ %s ] } } }`, strings.Join(tag, ","))
	var data M
	if data, err = e.do("GET", e.findAPI, []byte(json)); err != nil {
		return data, err
	}

	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()

	//重新封装数据，使之更合理
	var outData []map[string]interface{}

	//total
	var total float64
	if hitsInterface, ok := data["hits"]; ok {
		hits := hitsInterface.(map[string]interface{})
		total = hits["total"].(map[string]interface{})["value"].(float64)
		for _, hit := range hits["hits"].([]interface{}) {
			hitM := hit.(map[string]interface{})
			row := hitM["_source"].(map[string]interface{})
			row["id"] = hitM["_id"]
			outData = append(outData, row)
		}
	}
	if e.page == 0 {
		return outData, nil
	}

	//计算总页数
	var pageNum float64
	if e.pageSize != 0 {
		pageNum = math.Ceil(total / float64(e.pageSize))
	}

	return M{
		"data":      outData,
		"total":     total,
		"page":      e.page,
		"page_size": e.pageSize,
		"page_num":  pageNum,
	}, err
}

//Get 查找一条数据 通常和 Where("id=?",id)配合
func (e *Est) Get() (m map[string]interface{}, err error) {
	//判断是否有ID，ID参数通过where方法传递
	if e.table == "" {
		return m, errors.New(`table不存在，请使用SetTable("tableName")`)
	}
	//判断是否有ID，ID参数通过where方法传递
	if e.id == "" {
		return m, errors.New(`ID不存在，请使用Where("id=?",1)来传递`)
	}
	e.findAPI = fmt.Sprintf("%s://%s:%d/%s/_doc/%s",
		e.protocol,
		e.host,
		e.port,
		e.table,
		e.id,
	)
	var mJSON []byte
	var data M
	if data, err = e.do("GET", e.findAPI, mJSON); err != nil {
		return data, err
	}

	defer func() {
		e.clearData() //该处理的已经处理完了，可以清空临时内部数据了，防止est对象下一次调用时，产生脏数据
	}()

	//处理数据
	if outData, ok := data["_source"]; ok {
		m = outData.(map[string]interface{})
		m["id"] = e.id
	}

	return m, err
}

//do 执行请求,内部方法
func (e *Est) do(method string, api string, data []byte) (m M, err error) {
	//获取BASE64编码后的用户密码
	if e.auth == "" {
		var base64Encode = base64.StdEncoding.EncodeToString([]byte(e.user + ":" + e.password))
		e.auth = fmt.Sprintf("Basic %s", base64Encode)
	}
	var param = bytes.NewReader(data)
	var request = new(http.Request)
	if request, err = http.NewRequest(method, api, param); err != nil {
		return m, err
	}
	request.Header.Add("Content-Type", "application/json")
	request.Header.Add("Authorization", e.auth)
	var response = new(http.Response)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer func() {
		if response != nil && response.Body != nil {
			response.Body.Close()
		}
	}()
	if response == nil || response.Body == nil {
		return m, errors.New("elasticsearch host error")
	}

	//读取结果，这里用io.copy防止内存爆掉
	var buffer bytes.Buffer
	if _, err = io.Copy(&buffer, response.Body); err != nil {
		return m, err
	}

	//byte转map
	if err = json.Unmarshal(buffer.Bytes(), &m); err != nil {
		return m, err
	}
	return m, err
}
