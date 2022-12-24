#/bin/bash
echo $1 > VERSION
sed -i -e "s/.*buildVersion             = \"*.*/buildVersion =              \"$1\"/" ./connection.go
go fmt ./...
