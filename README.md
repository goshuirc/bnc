# GoshuBNC

GoshuBNC is an experimental IRC bouncer written in Go. It's designed to be simple to setup and feel fairly similar to ZNC when using it from the IRC side.

---

[![Go Report Card](https://goreportcard.com/badge/github.com/goshuirc/bnc)](https://goreportcard.com/report/github.com/goshuirc/bnc)
[![Freenode #goshuirc](https://img.shields.io/badge/Freenode-%23goshuirc-1e72ff.svg?style=flat)](https://www.irccloud.com/invite?channel=%23goshuirc&hostname=irc.freenode.net&port=6697&ssl=1)
<!--[![Download Latest Release](https://img.shields.io/badge/downloads-latest%20release-green.svg)](https://github.com/goshuirc/bnc/releases/latest)-->

---

This project adheres to [Semantic Versioning](http://semver.org/). For the purposes of versioning, we consider the "public API" to refer to the configuration files, CLI, and database format.

## Features

- Simple setup process and configuration files.
- It can connect up and be used!
- Not much else right now.

## Installation From Source

```sh
go build bnc.go
cp default-bnc.yaml bnc.yaml
vim bnc.yaml  # modify the config file to your liking
./bnc init
./bnc start
```

---

Parts of this project are based on code from the [Oragono](https://github.com/oragono/oragono)/[Ergonomadic](https://github.com/edmund-huber/ergonomadic) projects.
