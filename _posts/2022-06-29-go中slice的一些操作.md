---
layout: post
title: go中slice的一些操作
date: 2022-06-29
tags: go设计与实现
---

## intro

切片表示一个具有相同数据类型元素的的序列，切片的长度可变，通常写成[]T，其中元素的类型都是T。

切片用来访问数组的部分或全部元素，这个数组称为切片的底层数组。切片主要有三个属性：指针、长度和容量，指针指向切片的第一个元素，长度是指切片中元素的大小，而容量是指切片第一个元素到底层数组的最后一个元素间元素的个数。

## 切片的一些操作

切片的操作主要通过append,copy和切片操作符（s[i:j],其中 0<i<j<cap(s)）来完成，这里介绍一下切片常用的操作技巧和对数组应用切片操作时需要注意的问题。

1、切片常用操作技巧

（1)拼接两个切片

```go
 a = append(a, b...)  // 拼接切片a和b
```

(2)复制一个切片

```go
b = append([]T(nil), a...)
b = append(a[:0:0], a...)
```

(3)删除切片的第i~第j-1个元素([i,j))

```go
a[i:j]a = append(a[:i], a[j:]...) // 从a中删除
```

如果切片的元素是指针或者具有指针成员的结构体，需要避免内存泄露问题，此时需要修改删除切片元素的代码如下：

```go
for k, n := len(a)-j+i, len(a); k < n; k++ {
    a[k] = nil // 或该类型的零值}
a = a[:len(a)-j+i]
```

(4)删除第i个元素

```go
a = append(a[:i], a[i+1:]...) // 删除切片a的第i个元素
```

同样的，为了避免内存泄露

```go
copy(a[i:], a[i+1:])
a[len(a)-1] = nil // or the zero value of Ta = a[:len(a)-1]
```

 (5)弹出切片最后一个元素，即出队列尾(pop back)

```go
x, a = a[len(a) - 1], a[:len(a)-1]
```

(6)弹出切片第一个元素，即出队列头(pop)

```go
x, a = a[0], a[1:]
```

(7)在第i个元素前插入一个切片

```go
a = append(a[:i], append(b, a[i:]...)...)  // a[:i] 和a[i:]中间插入切片b
```

 (8)切片乱序(Go 1.10以上)

```go
for i := len(a) - 1; i > 0; i-- {
    j := rand.Intn(i + 1) // 生成一个[0,i+1)区间内的随机数
    a[i], a[j] = a[j], a[i]
}
```

2、切片操作符合Go语言中的可寻址性

首先简单介绍一下“可寻址性”，简单来说“可寻址性”是指如果一个对象可以应用取地址操作符&,那么这个对象就可以认为是可寻址的。

在使用切片的时候，对于数组、指向数组的指针或者切片s, 表达式s[low:high]构造了一个新的切片。不过经常会被忽略的一点是，如果**对一个数组进行切片操作，这个数组必须是可寻址的**，对于指向数组的指针或切片进行切片操作，则没有"可寻址性"的要求。

举例如下：

```go
a := [2]int{1,2}[:] // error,不能对不可寻址的数组进行切片操作。//output: invalid operation [2]int literal[:] (slice of unaddressable value)/* 对指向数组的指针进行切片操作 
*/func test() *[2]int{    
    return &[2]int{1,2}
}
b := test()[:] // succeed，可以对指向数组的指针进行切片操作/* 对切片进行切片操作 *
/func testSlice() []int {    
    return []int{1,2}
}
d := testSlice()[:] // succeed, 可以对切片进行切片操作。
```

03

切片作为参数在函数中传递

切片是一种引用类型，在64位架构的机器上，一个切片需要24个字节的内存：指针字段、长度字段和容量字段分别需要8字节，因此在函数中直接传递一个切片变量效率是非常高的，但是也正因为切片是引用类型，当函数使用切片作为形参变量的时候，函数内变量的改变可能会影响到函数外变量的值，比如下面这个例子：

```go
func main() {
    s1 := []string{"A", "B", "C"}
    fmt.Printf("before foo function, s1 is \t%v\n", s1)
    foo(s1)
    fmt.Printf("after foo function, s1 is \t%v", s1)
}
func foo(s []string) 
{
    s[0] = "New"
}
```

输出为:

```shell
before foo function, s1 is      [A B C]
after foo function,  s1 is      [New B C]
```

可以看到，函数foo中对切片s1的修改，确实影响到了函数外s1的值。但是在另外一些情况下，函数内对切片变量的改变却不会影响函数外的切片变量，还是看一个例子：

```go
func main() {
    s1 := []string{"A", "B", "C"}
    fmt.Printf("before foo function, s1 is \t%v\n", s1)
    foo(s1)
    fmt.Printf("after foo function, s1 is \t%v", s1)——
}
func foo(s []string) {
    s = append(s, "New")
}
```

输出为：

```go
before foo function, s1 is      [A B C]
after foo function,  s1 is      [A B C]
```

s1的值虽然在函数中改变，但是在函数外s1的值却没有变化。

那么，在函数中传递切片变量的时候，什么时候会影响外部变量，什么时候不会影响外部变量呢？其实可以这样理解：切片的标头值是一个指向底层数组的指针，当切片作为实参传递到函数中的时候，这个指针的值会复制给函数中的形参，即函数的实参和形参是共享同一个底层数组的，因此只要在函数中涉及到对底层数组值的修改，都会影响到函数外切片的值。

再举一个例子如下：

```go
func main() {
    arr := [5]string{"A", "B", "C", "D", "E"}
    s1 := arr[0:4]
    s2 := arr[2:4]
    fmt.Printf("before foo function, s2 is \t%v\n", s2)
    foo(s1)
    fmt.Printf("after foo function, s2 is \t%v", s2)
}
func foo(s []string) {
    s[2] = "NEW"
}
```

在这个例子里面，s1，s2 共享同一个底层数组，在foo()函数中，我们仍然修改s1的一个值，可以看到输出如下：

```shell
before foo function, s2 is      [C D]
after foo function,  s2 is      [NEW D]
```

s2的值因为s1对底层数组的修改，自身的值也被改变了。

## summ

在函数中传递切片变量的时候，**如果函数通过切片修改了底层数组的值，那么函数外指向该底层数组的切片的值也会被改变**，在Go中向函数传递切片变量的时候，需要特别注意这一点。

事实上，在Go语言中，所有的引用类型（切片、字典、通道、接口和函数类型），其标头值都包含一个指向底层数组的指针，因此通过复制来传递引用类型的值的副本，本质上就是在共享底层数据结构。