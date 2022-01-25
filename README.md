# Github Mining
Github mining scripts used for the paper:
> Mining the Usage of Reactive Programming APIs: A Mining Study on GitHub and Stack Overflow.

## Execution
### Requirements
Most of the scripts utilizes Golang (mainly) and Nodejs and they have be executed the following versions:
* Go (tested with v1.17.5)
* Node.js (tested with v14.17.5)

### Execution
The Go scripts are available under the `/cmd` folder.

```sh
go run cmd/operator-search/main.go
```
:computer: Script to search for the Rx operators.

:floppy_disk: After execution, the result is available at `assets/operators-search`.
```sh
go run cmd/repo-retrieval/main.go
```
:computer: Script to retrieve the repositories to be mined.

:floppy_disk: After execution, the result is available at `assets/repo-retrieval`.
```sh
go run cmd/repo-search/main.go
```
:computer: Script to search for repositories using selected rx libraries e save that information in a file, so repo-retrieval can proceed.

:floppy_disk: After execution, the result is available at `assets/repo-search`.
```sh
go run cmd/repo-summary/main.go
```
:computer: Script to create a summary of all rx distribution, including their total of dependent repositories, those with 0 stars and those with >=10 stars.

:floppy_disk: After execution, the result is available at `assets/repo-search`.

---

The Nodejs scripts, in turn, are available under the `/scripts/charts` folder. They were utilized post mining to generate charts and
data (CSV). Their results are available at `/scripts/charts/results`.

```sh
node generate-similarity
```
:computer: Script to generate charts and data related to RQ3.

```sh
node generate_frequencies
```
:computer: Script to generate charts and data related to the frequencies presented in RQ1.

```sh
node generate_utilization
```
:computer: Script to generate a chart showing the percentage of utilization when combine all frequencies from operators of the three studied libraries (not in the final version).

```sh
node generate_utilization_distribution
```
:computer: Script to generate a chart showing the percentage of utilization of the operators in each Rx library: RxJava, RxJS, and RxSwift.
