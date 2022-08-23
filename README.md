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
