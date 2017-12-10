#/bin/bash

mkdir ~/peerstore1
mkdir ~/peerstore2

echo "user1 sharing with user2" > ~/peerstore1/user1.txt
echo "user2 only" > ~/peerstore2/user2.txt

# create and register user1
./release/peerstore_client-latest-linux-amd64 -peerAddr :3001 -localPath ~/peerstore1/ -operation backup -peerKeyFile .peerstore/3001/publickey.pem  -selfKeyFile .peerstore/user1.pem
# create and register user2
./release/peerstore_client-latest-linux-amd64 -peerAddr :3001 -localPath ~/peerstore2/ -operation backup -peerKeyFile .peerstore/3001/publickey.pem  -selfKeyFile .peerstore/user2.pem

# attempting to get user1's text file using the user2 private key
./release/peerstore_client-latest-linux-amd64 -filedest ~/user1.txt.restored -peerAddr :3001 -filename ~/peerstore1/user1.txt -operation getfile -peerKeyFile .peerstore/3001/publickey.pem -selfKeyFile .peerstore/user2.pem
# this will fail, as user2 does not have permission for user1's file

# grant user2 access to user1.txt
./release/peerstore_client-latest-linux-amd64 -peerAddr :3001 -filename ~/peerstore1/user1.txt -operation share -peerKeyFile .peerstore/3001/publickey.pem  -selfKeyFile .peerstore/user1.pem -shareWithKeyFile .peerstore/user2.pem

# user2 "getfile" user1.txt which was shared
./release/peerstore_client-latest-linux-amd64 -filedest ~/user1.txt.restored -peerAddr :3001 -filename ~/peerstore1/user1.txt -operation getfile -peerKeyFile .peerstore/3001/publickey.pem -selfKeyFile .peerstore/user2.pem
