---
title: GC Test
keywords: [higress, gc test]
description: use to test the gc of tinygo
---

## Description

The `gc-test` plugin is used to test whether there are memory leaks in TinyGO's GC mechanism.

This plugin should not be used in production.

## Configuration Fields

| Name        | Type            | Requirement | Default Value | Description                                          |
| ----------- | --------------- | --------    | ------        | ---------------------------------------------------- |
| `bytes`     | Number          | Required    | -             | Number of bytes allocated per-request                |


## How to

The plugin will response the stats of memory as follows:

```bash
{"Sys": 15073280,"HeapSys": 10682368,"HeapIdle": 139264,"HeapInuse": 0,"HeapReleased": 0}
```

We can use bench tools to test whether the `HeapSys` field keeps growing, and then we can determine whether a memory leak has occurred.
