# gitbitex-service-group

<p align="center"><img width="40%" src="https://getbitex.oss-cn-beijing.aliyuncs.com/projects/image/logo.svg" /></p>

[![Go Report Card](https://goreportcard.com/badge/github.com/gitbitex/gitbitex-spot)](https://goreportcard.com/report/github.com/gitbitex/gitbitex-spot)

GitBitEx is an open source cryptocurrency exchange.

## Architecture
<p align="center"><img width="100%" src="https://oooooo.oss-cn-hangzhou.aliyuncs.com/gitbitex.png?v=1" /></p>

## Demo
We deployed a demo version on a cloud server (2 Cores CPU 4G RAM). All programs run on this server. include (mysql,kafka,redis,gitbitex-spot,nginx,web...).

https://gitbitex.com:8080/trade/BTC-USDT

## Dependencies
* MySql (**BINLOG[ROW format]** enabled)
* Kafka
* Redis

## Install
### Server
* git clone https://github.com/gitbitex/gitbitex-spot.git
* Create database and make sure **BINLOG[ROW format]** enabled
```
/etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
server-id=1      
log-bin = mysql-bin 
```

* Execute ddl.sql
```
/etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
sql-mode=ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION
```

* Modify conf.json
```
cp conf_example.json conf.json
```

* Build 
    
##### legacy:
```
make legacy_clean
make legacy
```

##### recent:
```
make clean
make all
```

* Run 
##### legacy:
```
./gitbitex-spot
```

##### recent:
```
./billing_processor 
./binlog_processor 
./matching_engine 
./ordering_processor 
./trading_processor 
./pushing_server 
./rest_server
```


### Web
* git clone https://github.com/gitbitex/gitbitex-web.git
* Run `npm install`
* Run `npm start`
* Run `npm run build` to build production


## Questions?
Please let me know if you have any questions. You can submit an issue or send me an email (greensheng2001@gmail.com) or Telegram (https://t.me/greensheng)

## Contributing
This project welcomes contributions and suggestions and we are excited to work with the power user community to build the best exchange in the world
