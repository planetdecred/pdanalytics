# pdanalytics

[![Build Status](https://img.shields.io/travis/decred/dcrdata.svg)](https://travis-ci.org/planetdecred/pdanalytics)
[![Go Report Card](https://goreportcard.com/badge/github.com/decred/dcrdata)](https://goreportcard.com/report/github.com/planetdecred/pdanalytics)
[![ISC License](https://img.shields.io/badge/license-ISC-blue.svg)](http://copyfree.org)

pdanalytics contains modules for exploring the decred blockchain and collecting additional info about the decred cryptocurrency like ticker and orderbook data from various exchanges. Each module can be enabled/disabled giving the user the ability to run a light-weight version of the program at will. 

## Requirements
To run **pdanalytics** on your machine you will need the following to be setup.
- `Go` 1.13
- `Postgresql` (Optional, depending on the activated modules)
- `Nodejs.` Node.js is only used as a build tool, and is not used at runtime
- `Dcrd` (Optional, depending on the activated modules)

## Setting Up Pdanalytics 
### Step 1. Installations
**Install Go**
* Minimum supported version is 1.13. Installation instructions can be found [here](https://golang.org/doc/install).

**Install Postgrsql**
* Postgrsql is a relational DBMS used for data storage. Download and installation guide can be found [here](https://www.postgresql.org/download/)
* *Quick start for  Postgresql*

    If you have a new postgresql install and you want a quick setup for pdanalytics, you can start `postgresql command-line client`(It comes with the installation) with...
    
    ***Linux***
  -  `sudo -u postgres psql` or you could `su` into the postgres user and run `psql` then execute the sql statements below to create a user and database.
    
    ***Windows***
  - Just open the command line interface and type `psql` then execute the sql statements below to create a user and database.
```sql
    CREATE USER {username} WITH PASSWORD '{password}' CREATEDB;
    CREATE DATABASE {databasename} OWNER {username};
```
**Install Nodejs**
* Instructions on how to install `Nodejs` can be found [here](https://nodejs.org/en/download/)

**Install Dcrd**
* Running `dcrd` synchronized to the current best block on the network.
* Download the **decred** release binaries for your operating system from [here](https://github.com/decred/decred-binaries/releases). Check under **Assets**.
* The binary contains other decred packages for connecting to the decred network. 
* Extract **dcrd** Only, [go here](https://docs.decred.org/wallets/cli/cli-installation/) to learn how to setup and run decred binaries.

### Step 2. Getting the source code
- Clone the *pdanalytics* repository. It is conventional to put it under `GOPATH`, but
  this is no longer necessary with go module.

```bash
  git clone https://github.com/planetdecred/pdanalytics
 ```

### Step 3. Building the source code.
* If you cloned to $GOPATH, set the `GO111MODULE=on` environment variable before building.
Run `export GO111MODULE=on` in terminal (for Mac/Linux) or `setx GO111MODULE on` in command prompt for Windows.
* `cd` to the cloned project directory and run `go build` or `go install`.
Building will place the `pdanalytics` binary in your working directory while install will place the binary in $GOPATH/bin.

#### Building http front-end
* From your project directory, `cd` into the `web` folder and run `npm install` when its done installing packages, 
run `npm run build`.

### Step 4. Configuration
`pdanalytics` can be configured via command-line options or a config file located in the home directory. Start with the sample config file:
```sh
cp sample-pdanalytics.conf ~/.pdanalytics/pdanalytics.conf
```
Then edit `~/.pdanalytics/pdanalytics.conf` with your postgres settings. See the output of `pdanalytics --help`
for a list of all options and their default values.

## Running pdanalytics
To run *pdanalytics*, use...
- `pdanalytics` on your command line interface to create database table, fetch data and store the data and launch the http web server. The web server can be disabled by setting `--nohttp`
- You can perform a reset by running with the `-R` or `--reset` flag.
- Run `pdanalytics -h` or `pdanalytics help` to get general information of commands and options that can be issued on the cli.
- Use `pdanalytics <command> -h` or   `pdanalytics help <command>` to get detailed information about a command.

## Contributing
See the CONTRIBUTING.md file for details. Here's an overview:

1. Fork this repo to your github account
2. Before starting any work, ensure the master branch of your forked repo is even with this repo's master branch
3. Create a branch for your work (`git checkout -b my-work master`)
4. Write your codes
5. Run the code linter
```bash
golangci-lint run --deadline=10m --disable-all \
    --enable govet \
    --enable staticcheck \
    --enable gosimple \
    --enable unconvert \
    --enable ineffassign \
    --enable goimports \
    --enable misspell
```
6. Build and test your changes
```bash
go build -o pdanalytics -v -ldflags \
    "-X github.com/planetdecred/pdanalytics/version.appPreRelease=beta \
     -X github.com/planetdecred/pdanalytics/version.appBuild=`git rev-parse --short HEAD`"
```

7. Commit and push to the newly created branch on your forked repo
8. Create a [pull request](https://github.com/planetdecred/pdanalytics/pulls) from your new branch to this repo's master branch
