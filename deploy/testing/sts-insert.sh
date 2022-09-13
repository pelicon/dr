#!/bin/bash

function rand(){
    min=$1
    max=$(($2-$min+1))
    num=$(date +%s%N)
    echo $(($num%$max+$min))
}

function insert(){  
    user=$(date +%s%N | md5sum | cut -c 1-9)
    age=$(rand 1 100)

    sql="INSERT INTO test.users(user_name, age)VALUES('${user}', ${age});"
    echo -e ${sql}

    kubectl exec sts/dr-mysql-sts -- mysql -uroot -pdangerous -e "${sql}"

}

kubectl exec sts/dr-mysql-sts -- mysql -uroot -pdangerous -e "CREATE DATABASE IF NOT EXISTS test;"
kubectl exec sts/dr-mysql-sts -- mysql -uroot -pdangerous -e "CREATE TABLE IF NOT EXISTS test.users(user_name VARCHAR(10) NOT NULL,age INT UNSIGNED)ENGINE=InnoDB DEFAULT CHARSET=utf8;"

while true;do
    insert
    sleep 1
done
