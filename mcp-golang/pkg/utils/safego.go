package utils

func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				println("[Error::Go] recover from panic", r)
			}
		}()
		fn()
	}()
}
