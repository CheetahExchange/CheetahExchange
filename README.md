# CheetahExchange

Forked from [gitbitex-spot](https://github.com/gitbitex/gitbitex-spot)

## Install Dependent Infrastructures
* MySql (make sure **BINLOG[ROW format]** enabled)
```
sudo apt-get install mysql-server
```

```
/etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
server-id=1      
log-bin = mysql-bin 
```

```
# Mysql 5.x
/etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
sql-mode=ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION

# Mysql 8.x
/etc/mysql/mysql.conf.d/mysqld.cnf
[mysqld]
sql-mode=ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION
```

* Zookeeper
```
bash bin/zkServer.sh start
```

* Kafka
```
bash bin/kafka-server-start.sh config/server.properties
```

* Redis
```
sudo apt-get install redis-server
```


## Install Golang Compiler

* [Golang](https://go.dev/doc/install)


## Build Server

* Clone Repo
```
git clone https://github.com/CheetahExchange/CheetahExchange
cd CheetahExchange
```

* Build 
```
make clean
make
```

## Run Server
* Modify conf.json
```
cp conf_example.json conf.json
```

* Run
```
./spot-core
./spot-rest
./spot-pushing
```
