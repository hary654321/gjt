#! /bin/bash
ulimit -n 50000
monitorM=`ps -ef | grep gjt| grep -v grep | wc -l ` 
if [ $monitorM -eq 0 ] 
then
	echo "gjt is not running, restart gjt"
	gjt 800 >>m.log &
else
	echo "gjt is running"
fi

