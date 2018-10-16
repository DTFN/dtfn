is sh new more accounts of eth
# $1 is the number of accounts, $2 is the PWD
for ((i=0;i<$1;i++))
do
  geth --password $PWD/pwd --datadir $PWD account new 
done


