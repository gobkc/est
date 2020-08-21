1.初始化
	es := est.NewEst()
	es = es.SetHost("xxx").SetPort(9200).SetProtocol("http").SetUser("xxx").SetPassword("xxx")

2.新增数据

	if data, err := es.Table("table1").Add(est.M{"aaa": "排序4", "ccc": "444","sort":47}); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

3.修改数据

	if data, err := es.Table("table1").Where("id=?", "ev-HBXQBkU4LsT8FrPCa").Save(est.M{"aaa": "1998"}); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

4.递增字段 +5

	if data, err := es.Table("table1").Where("id=?", "iP-JCnQBkU4LsT8FhfAm").SetInc("sort",5); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

5.递减字段 -5

	if data, err := es.Table("table1").Where("id=?", "iP-JCnQBkU4LsT8FhfAm").SetDec("sort",5); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

    删除指定ID的数据(目前删除数据只能通过指定ID，不能批量删除或根据其他条件删除)

	if data, err := es.Table("table1").Where("id=?", "ef_eBHQBkU4LsT8F6_AX").Delete(); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

6.查询数据(注意,使用了SetPage方法，返回的数据中包含有，记录数，页数等额外信息「MAP」，不使用则返回的是一个数据列表)

	if data, err := es.Table("table1").Where("aaa=?", "张三").SetPage(1).Find(); err != nil {
		log.Println("err：", err.Error())
	} else {
		log.Println("数据：", data)
	}

7.查询一条数据

    if data, err = es.Table("finance").Where("id=?", "ef_eBHQBkU4LsT8F6_AX").Get(); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
