---
layout: post
title: channel的底层实现
date: 2022-07-20
tags:  go设计与实现 
---


## 表现形式
channel跟string或slice有些不同，它在栈上只是一个指针，实际的数据都是由指针所指向的堆上面。
跟channel相关的操作有：初始化/读/写/关闭。channel未初始化值就是nil，未初始化的channel是不能使用的。下面是一些操作规则：

读或者写一个nil的channel的操作会**永远阻塞**。
读一个关闭的channel会立刻返回一个**channel元素类型的零值**。
写一个关闭的channel会导致panic。


channel数据结构

``go
struct    Hchan
{
    uintgo    qcount;            // 队列q中的总数据数量
    uintgo    dataqsiz;        // 环形队列q的数据大小
    uint16    elemsize;
    bool    closed;
    uint8    elemalign;
    Alg*    elemalg;            // interface for element type
    uintgo    sendx;            // 发送index
    uintgo    recvx;            // 接收index
    WaitQ    recvq;            // 因recv而阻塞的等待队列
    WaitQ    sendq;            // 因send而阻塞的等待队列
    **Lock;**  //有锁
}
``
让我们来看一个Hchan这个结构体。其中一个核心的部分是存放channel数据的环形队列，由qcount和elemsize分别指定了队列的容量和当前使用量。dataqsize是队列的大小

如果是带缓冲区的chan，则缓冲区数据实际上是紧接着Hchan结构体中分配的。
c = (Hchan*)runtime.mal(n + hint*elem->size);

另一个重要部分就是recvq和sendq两个链表，一个是因读这个通道而导致阻塞的goroutine，另一个是因为写这个通道而阻塞的goroutine。如果一个goroutine阻塞于channel了，那么它就被挂在recvq或sendq中。



Hchan结构如下图所示: 

读写channel操作
先看写channel的操作，基本的写channel操作，在底层运行时库中对应的是一个runtime.**chansend**函数。


这个函数首先会**区分是同步还是异步**。同步是指chan是不带缓冲区的，因此可能写阻塞，而异步是指chan带缓冲区，只有缓冲区满才阻塞。

在同步的情况下，由于channel本身是不带数据缓存的，这时首先会查看Hchan结构体中的recvq链表时否为空，即是否有因为读该管道而阻塞的goroutine。如果有则可以正常写channel，否则操作会阻塞。


在异步的情况，如果缓冲区满了，也是要将当前goroutine挂在sendq队列中，表示因写channel而阻塞。否则也是先看有没有recvq链表是否为空，有就唤醒。

跟同步不同的是在channel缓冲区不满的情况，这里不会阻塞写者，而是将数据放到channel的缓冲区中，调用者返回。

读channel的操作也是类似的，对应的函数是runtime.chansend。一个是收一个是发，基本的过程都是差不多的。

需要注意的是几种特殊情况下的通道操作--空通道和关闭的通道。

空通道是指将一个channel赋值为nil，或者定义后不调用make进行初始化。按照Go语言的语言规范，读写空通道是永远阻塞的。

读一个关闭的通道，永远不会阻塞，会返回一个通道数据类型的零值。这个实现也很简单，将零值复制到调用函数的参数ep中。写一个关闭的通道，则会panic。关闭一个空通道，也会导panic。

select的实现
**select-case中的chan操作编译成了if-else。比如：**

select {
case v = <-c:
        ...foo
default:
        ...bar
}
会被编译为:

if selectnbrecv(&v, c) {
        ...foo
} else {
        ...bar
}
类似地

select {
case v, ok = <-c:
    ... foo
default:
    ... bar
}
会被编译为:

if c != nil && selectnbrecv2(&v, &ok, c) {
    ... foo
} else {
    ... bar
}


在Go的语言规范中，select中的case的执行顺序是随机的，而不像switch中的case那样一条一条的顺序执行。那么，如何实现随机呢？

select和case关键字使用了下面的结构体：

struct    Scase
{
    SudoG    sg;            // must be first member (cast to Scase)
    Hchan*    chan;        // chan
    byte*    pc;            // return pc
    uint16    kind;
    uint16    so;            // vararg of selected bool
    bool*    receivedp;    // pointer to received bool (recv2)
};

struct    Select
{
    uint16    tcase;            // 总的scase[]数量
    uint16    ncase;            // 当前填充了的scase[]数量
    uint16*    **pollorder;**        // case的poll次序
    Hchan**    lockorder;        // channel的锁住的次序
    **Scase**    scase[1];        // 每个case会在结构体里有一个Scase，顺序是按出现的次序
};
每个select都对应一个Select结构体。在Select数据结构中有个Scase数组，记录下了每一个case，而Scase中包含了Hchan。然后**pollorder数组将元素随机排列**，这样就可以将Scase乱序了。
