for ((i=1; i<=100; i ++))
do
    echo `curl --proxy https://localhost:8081 --proxy-insecure --proxy-header "Naiba: lifelonglearning" http://api.ip.la`
done