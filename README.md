	es := est.NewEst()
	es = es.SetHost("xxx").SetPort(9200).SetProtocol("http").SetUser("xxx").SetPassword("xxx")
	//if data, err := es.Table("table1").Add(est.M{"aaa": "排序4", "ccc": "444","sort":47}); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}

	//if data, err := es.Table("table1").Where("id=?", "ev-HBXQBkU4LsT8FrPCa").Save(est.M{"aaa": "1998"}); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}

	//递增5
	//if data, err := es.Table("table1").Where("id=?", "iP-JCnQBkU4LsT8FhfAm").SetInc("sort",5); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}

	////递减5
	//if data, err := es.Table("table1").Where("id=?", "iP-JCnQBkU4LsT8FhfAm").SetDec("sort",5); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}

	//if data, err := es.Table("table1").Where("id=?", "ef_eBHQBkU4LsT8F6_AX").Delete(); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}

	//if data, err := es.Table("table1").Where("aaa=?", "张三").SetPage(1).Find(); err != nil {
	//	log.Println("err：", err.Error())
	//} else {
	//	log.Println("数据：", data)
	//}
