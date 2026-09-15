#!/bin/sh

# 1. View current prefix
./bin/vault prefix

# 2. Set default prefix (updates .env automatically)
./bin/vault prefix myserver

# 3. Write a secret (uses prefix; generates random Star Trek name if value is omitted)
./bin/vault write rabbitmq/username "guest"
./bin/vault write rabbitmq/password "testpassword"
./bin/vault write rabbitmq/host "192.168.10.10"
./bin/vault write rabbitmq/port "5672"

./bin/vault write mysql {"username":"guest","password":"muysecreto","host":"192.168.10.10","port":"3306"}

# 4. Read a secret
./bin/vault read rabbitmq/username
./bin/vault read rabbitmq/password
./bin/vault read rabbitmq/host
./bin/vault read rabbitmq/port

./bin/vault read mysql
./bin/vault read mysql.username
./bin/vault read mysql.password
./bin/vault read mysql.host
./bin/vault read mysql.port

# 5. List keys under the prefix
./bin/vault list

# 6. Delete a secret
./bin/vault delete rabbitmq/username
./bin/vault delete rabbitmq/password
./bin/vault delete rabbitmq/host
./bin/vault delete rabbitmq/port

./bin/vault list mysql
./bin/vault delete mysql.username
./bin/vault list mysql
./bin/vault delete mysql

./bin/vault delete -r myserver


