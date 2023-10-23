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

The plugin will log memstats after allocating memory per-request as follows:

```bash
[gc-test] MemStats Sys:67633152, HeapSys:63176704, HeapIdle:10653696, HeapInuse:0, HeapReleased:0
```

We can use bench tools to test whether the `HeapSys` field keeps growing, and then we can determine whether a memory leak has occurred.
