[![Coverage Status](https://coveralls.io/repos/github/danlapid/oneway-filesync/badge.svg?branch=main)](https://coveralls.io/github/danlapid/oneway-filesync?branch=main)
go run oneway-filesync/cmd/receiver \
go run oneway-filesync/cmd/sender



DB Reader:
    output: paths
FileReader
    input: paths
    output: chunks
 ->
FEC
    input: chunks
    output: chunks
 ->
Rate Limiter
    input: chunks
    output: chunks
 ->
Socket Sender
    input: chunks


Socker reader
    output: chunks
 ->
DeFEC
    input: chunks
    output: chunks
 ->
File Writer
    input: chunks:
    output: Finished files


psql -h 127.0.0.1 -p 5432 -U postgres -d postgres -c "SELECT * FROM files"
psql -h 127.0.0.1 -p 5432 -U postgres -d postgres -c "DELETE FROM files"
TODO: different db for sent and received
