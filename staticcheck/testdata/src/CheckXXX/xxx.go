package pkg

var sink int

func fn1(x interface{}) {
	_ = func() {
		switch y := x.(type) {
		case int:
			println(y)
		case string:
			println(y)
		default:
			println(y)
		}
	}
}

func fn2() {
	sink = 1
}
