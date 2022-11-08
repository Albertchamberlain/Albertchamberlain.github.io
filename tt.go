package main

import "fmt"

func main() {
	var T, i, n, j, k int
	fmt.Scan(&T)
	for k = 0; k < T; k++ {
		flag := 0
		fmt.Scan(&n)
		a := make([]int, n)
		for i = 0; i < n; i++ {
			for j := 0; j < n; j++ {
				fmt.Scan(&a[i][j])
			}
			if i > j && a[i][j] != 0 { //利用矩阵性质，即j>i的项不等于零来判断是否为上三角形矩阵
				flag = 1
			}
		}
		if flag == 0 {
			fmt.Println("YES")
		} else {
			fmt.Println("NO")
		}
	}
}
