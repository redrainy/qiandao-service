# 自定义签到服务

### 使用到的两个库
&nbsp;&nbsp;[UUID](https://github.com/pschlump/uuid) &nbsp;&nbsp;[GoTicker](https://github.com/aWildProgrammer/goticker)

### 添加接口
请求链接: `:8088/qiandao/addCheckPoint`

请求方式: `POST`

请求示例
```
{
  "website": "xx网站",
  "link": "https://www.xxx.com/qiandao",
  "method": "POST",
  "day": 28,
  "time": "20:10",
  "note": "xx网站晚上八点打卡",
  "cookie": "__cfduid=1594108105lastcheckfeed=260",
  "protocol": "application/x-www-form-urlencoded",
  "data": "button4=1;click=",
  "isTrue": 1
}
```
### 查询接口
请求链接: `:8088/qiandao/checkPoints`

请求方式: `GET`

返回示例
```
[
  {
    "uuid": "47e3e190-f69d-4dae-6be1-83ac2c71d05a",
    "website": "xx网站",
    "link": "https://www.xxx.com/qiandao",
    "method": "POST",
    "day": 0,
    "time": "20:10",
    "note": "xx网站每天晚上八点打卡",
    "cookie": "__cfduid=1594108105lastcheckfeed=260",
    "protocol": "application/x-www-form-urlencoded",
    "data": "button4=1;click=",
    "isTrue": 1,
    "isRequest": 1
  },
  {
    "uuid": "205047d0-b16d-4210-43e7-c4af4ffa03d8",
    "website": "xx网站",
    "link": "https://www.xxx.com/qiandao?xxx=xxx",
    "method": "POST",
    "day": 28,
    "time": "8:10",
    "note": "xx网站28号早上八点打卡",
    "cookie": "__cfduid=1594108105lastcheckfeed=260",
    "protocol": "application/x-www-form-urlencoded",
    "data": "button4=1;click=",
    "isTrue": 1,
    "isRequest": 0
  }
]
```
### 删除接口
请求链接: `:8088/qiandao/deleteCheckPoint`

请求方式: `POST`

请求示例
```
{
  "uuid": "205047d0-b16d-4210-43e7-c4af4ffa03d8"
}
```
### 修改接口
请求链接: `:8088/qiandao/updateCheckPoint`

请求方式: `POST`

请求示例
```
{
    "uuid": "205047d0-b16d-4210-43e7-c4af4ffa03d8",
    "website": "xx网站",
    "link": "https://www.xxx.com/qiandao?xxx=xxx",
    "method": "POST",
    "day": 27,
    "time": "18:10",
    "note": "xx网站27号下午六点打卡",
    "cookie": "__cfduid=1594108105lastcheckfeed=260",
    "protocol": "application/x-www-form-urlencoded",
    "data": "button3=1;click=",
    "isTrue": 1,
    "isRequest": 0
  }
```

### 文件结构:
```
  qiandao|
         |-qiandao_service
         |-conf|
         |     |-qiandao.json
         |
         ---------------
         |-qiandao|
         |        |07|-qiandao.json.01
         |        |  |-qiandao.json.02
         |        |  |-qiandao.json...
         |        |
         |        |08|-qiandao.json.01
         |        |  |-qiandao.json.02
         |        |  |-qiandao.json...
```


##### 主方法->[签到方法]每30分钟执行一次


### 数据结构
```
{   
  网站名称;
  签到链接;
  请求方式[GET,POST];
  签到时间;
  day[0(每天都签到),1-28];
  备注;
  cookie;
  请求协议[填写抓包content-type];
  请求参数[请以k=v方式存放,;分割];
  是否为真[0(否),1(是)];
}
```
