# Github Mining
Github mining scripts used for the paper:
> Mining the Usage of Reactive Programming APIs: A Mining Study on GitHub and Stack Overflow.

## Data
Under the folders in `/assets`, data either genereated by or collected for the scripts execution can be found. The table gives a brief description of each folder:

| Folder   | Description         |
| :------------- |:-------------|
| operators-search | Includes the results for the Rx libraries' operator search |
| operators | Includes JSON files consisting of Rx libraries' operators |
| repo-retrieval | Contains data about the GitHub repositories retrieved and processed |
| result-search | Contains information about each dependent repository with >= 10 stars|
| result-summary | Contains a summary of all rx distribution, including their total of dependent repositories, those with 0 stars and those with >=10 stars  |
| so-data | Stack Overflow data colleted through the Stack Mining scripts |

The file `Programming_Languages_Extensions.json` contains a list extensions used by several languages. In the paper, the following entries were utilized:
* Java (for RxJava)
* JSX, JavaScript, and TypeScript (for RxJS)
* Swift (for RxSwift)

As detailed in the paper, the `repo-retrieval` result do not provides the actual GitHub repositories given the size constraints to upload them here (even if they are compressed as tarball files). Instead, under each rx library folder inside `repo-retrieval` (e.g., `/assets/repo-retrieval/rxjava`), there is a file called `list_of_files.json` containing an array of objects with the following info that can be used to download the exactly same files (that must me place in a subfolder `/archives` relative to `list_of_files.json`):
| Entry   | Description         |
| :------------- |:-------------|
| owner | the owner of the repository |
| repoName | the repository name |
| repoFullName | the full repository name (i.e., with the ownwer concatenated) |
| branch | the default branch |
| fileName | the name of the tarball file |
| fileSize | the files' size in bytes |
| url | the url to download the tarball file with the SHA1 of the last commit already set |

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
:computer: Script to generate a chart showing the percentage of utilization when combine all frequencies from operators of the three studied libraries (not present in the paper only the percentage).

```sh
node generate_utilization_distribution
```
:computer: Script to generate a chart showing the percentage of utilization of the operators in each Rx library: RxJava, RxJS, and RxSwift.
