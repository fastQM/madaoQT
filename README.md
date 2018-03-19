The trading framework of the crypto assets


[Requirement]
1. go version >= 1.9.2
2. MongoDB

[Installation]
1. git clone https://github.com/maisuid/madaoQT
2. cd madaoQT
3. sh install.sh
4. cd www
5. bower install
6. cd ..
7. go run main.go
8. Open link: http://localhost:8080

[modules]
1. config
The global configuration of the system.

2. exchange
The API of the exchanges

3. mongo
The interface of the mongo to save to the tracking datas of the crypto-currency and the trading history.

4. server
The http server to handle the http requests from clients

5. task
We implment the trading strategies here. Each strategy is a task

6. www
The front websites and resources