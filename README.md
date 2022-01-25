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

### Scripts
The Go scripts are available under the `/cmd` folder. All of them are structured to be executed at the root of the repository.
Before execution of any Go script, one must run the following command in a terminal to install all the dependencies:
```sh
go mod tidy
```

**operator-search**

Script to search for the Rx operators.
```sh
go run cmd/operator-search/main.go
```
&ensp;:floppy_disk: After execution, the result is available at `assets/operators-search`.

---
**repo-retrieval**

Script to retrieve the repositories to be mined.
```sh
go run cmd/repo-retrieval/main.go
```
&ensp;:floppy_disk: After execution, the result is available at `assets/repo-retrieval`.

---
**repo-search**

Script to search for repositories using selected rx libraries e save that information in a file, so repo-retrieval can proceed.
```sh
go run cmd/repo-search/main.go
```
&ensp;:floppy_disk: After execution, the result is available at `assets/repo-search`.

---
**repo-summary**

Script to create a summary of all rx distribution, including their total of dependent repositories, those with 0 stars and those with >=10 stars.
```sh
go run cmd/repo-summary/main.go
```
&ensp;:floppy_disk: After execution, the result is available at `assets/repo-search`.

#### Nodejs scripts

The Nodejs scripts, in turn, are available under the `/scripts/charts` folder. They were utilized post mining to generate charts and
data (CSV). Their results are available at `/scripts/charts/results`.
Before execution of any Node script, one must run the following command in a terminal to install all the dependencies:
```sh
npm install
```
All the Node scripts should use `/scripts/charts` as the working directory.

##### generate-similarity
Script to generate charts and data related to RQ3.
```sh
node generate-similarity
```
&ensp;:floppy_disk: By the end of execution, the results are available in `/scripts/charts/results/similarity`.
The script produces many outputs and they can be generalized as:
<br/>&emsp;&emsp;&emsp;:white_medium_small_square: **frequencies\__[relevant topic]_\__[rx library]_.csv**: operators frequencies of the most relevant topics in the rx libraries analyzed. Usage frequencies acquired from Stack Overflow posts;
<br/>&emsp;&emsp;&emsp;:white_medium_small_square: **similarities\__[relevant topic]_\__[rx library]_.csv**: contains the operators(and their frequencies) of the most relevant topics according to the rx libraries analyzed in which their frequency placement matches the frequency placement of the rx operators' frequencies collected in GitHub projects;
<br/>&emsp;&emsp;&emsp;:white_medium_small_square: **similarities\__[ 'leastUsed' | 'mostUsed' ]_\__[relevant topic]_\__[rx library]_.csv**: close to the results above but considering the top least and most used operators and disconsidering their placement(order);
<br/>&emsp;&emsp;&emsp;:white_medium_small_square: **similarity.png**: percentage of similarity when comparing the operators (sorted by their frequency) of the most relevant topics and the operators (also sorted by their frequencies) found in GitHub repositories. This figure takes into account the order of appearance of each operator (their placement) according to their frequency;
<br/>&emsp;&emsp;&emsp;:white_medium_small_square: **similarity\__[ 'leastUsed' | 'mostUsed' ]_.png**: percentage of similarity when comparing the top least and most used (according to their frequencies) operators of the most relevant topics and the top least and most used operators (sorted by their frequencies) found in GitHub repositories. This figure does not take into account the order of appearance of each operator (their placement) according to their frequency. It only considers if there is a match between the lists of least or most used operators;

Where, _[relevant topic]_ = {Dependency Management, Introductory Questions, iOS Development} and _[rx library]_ = {RxJava, RxJS, RxSwift}

---
##### generate_frequencies
Script to generate charts and data related to the frequencies presented in RQ1.
```sh
node generate_frequencies
```
&ensp;:floppy_disk: By the end of execution, the results are available in `/scripts/charts/results/frequency`.
The script produces many outputs and they can be summarized as:
<br/>&emsp;&emsp;&emsp;:white_medium_small_square:**frequency\__[rx library]_\__[ 'topLeastUsed' | 'topMostUsed' ]_.png**: charts showing the top least and most used operator, according to their frequency, of each Rx library analyzed;
<br/>&emsp;&emsp;&emsp;:white_medium_small_square:**frequency\__[rx library]_.[ csv | json ]** - CSV and JSON files containing operator frequencies of each Rx library analyzed.

Where, _[rx library]_ = {RxJava, RxJS, RxSwift}

---
##### generate_utilization
Script to generate a chart showing the percentage of utilization when combine all frequencies from operators of the three studied libraries (not present in the paper only the percentage).
```sh
node generate_utilization
```
&ensp;:floppy_disk: By the end of execution, the result is placed in `/scripts/charts/results/utilization`

---
##### generate_utilization_distribution
Script to generate a chart showing the percentage of utilization of the operators in each Rx library: RxJava, RxJS, and RxSwift.
```sh
node generate_utilization_distribution
```
&ensp;:floppy_disk: By the end of execution, the result is placed in `/scripts/charts/results/utilization_perDistribution`
