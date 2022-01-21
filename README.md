# Github Mining
Github mining scripts used for the paper:
> Mining the Usage of Reactive Programming APIs: A Mining Study on GitHub and Stack Overflow.

## Requirements
Most of the scripts utilizes Golang (mainly) and Nodejs and they have be executed the following versions:
* Go version 1.17.5
* Node.js version 14.17.5

## Execution
The Go scripts are available under the `/cmd` folder

```go
go run cmd/operator-search/main.go
```
:computer: Script to search for the Rx operators.

:floppy_disk: After execution, the result is available at `assets/operators-search`.
```go
go run cmd/repo-retrieval/main.go
```
:computer: Script to retrieve the repositories to be mined.

:floppy_disk: After execution, the result is available at `assets/repo-retrieval`.
```go
go run cmd/repo-search/main.go
```
:computer: Script to search for repositories using selected rx libraries e save that information in a file, so repo-retrieval can proceed.

:floppy_disk: After execution, the result is available at `assets/repo-search`.
```go
go run cmd/repo-summary/main.go
```
:computer: Script to create a summary of all rx distribution, including their total of dependent repositories, those with 0 stars and those with >=10 stars.

:floppy_disk: After execution, the result is available at `assets/repo-search`.
