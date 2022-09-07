---
layout: post
title: protobuf+go+rpc体验
date: 2022-09-06
tags:  go设计与实现 
---

## Protobuf 简介
Protobuf类似于XML、JSON等数据描述语言，它可以通过自带的工具生成代码，并实现将结构化数据序列化的功能。
Protobuf中最基本的数据单元是message，是类似Go语言中结构体的存在。在message中可以嵌套message或其它的基础数据类型的成员。

## Protobuf 使用
```protobuf
syntax = "proto3"; //采用proto3的语法

package main;

message String {
    string value = 1;
}
```
message关键字定义一个新的String类型，生成的代码也即对应Go中的String结构体，因为String类型只有一个成员，因此String被编码时用1代替名字。

在平时开发遇到的JSON序列化时，一般通过成员的名字来绑定对应的数据。而Protobuf通过成员的唯一编号来绑定对应的数据，这样的有点就是Protobuf编码后数据的体积会比较小，
但是缺点就是不直观。

## 代码生成

1. 使用C++编写的插件有proto文件生成对应的Go代码
https://github.com/google/protobuf/releases
2. 安装针对Go语言的代码生成插件
   ```go
    go get -u github.com/golang/protobuf/protoc-gen-go
    ```

3. 然后通过命令生成相应的Go代码：
    ```shell
    protoc --go_out=. hello.proto
    ```
    go_out 参数告知protoc编译器去加载对应的protoc-gen-go工具，然后通过该工具生成代码，生成代码放到当前目录。
    如果报错 protoc-gen-go: unable to determine Go import path for "hello.proto"
    需要指定包路径  在syntax = "proto3"; 下面加一行 option go_package ="goadvance/";

