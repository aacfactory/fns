package metrics

func init() {
	for i := 0; i < 4; i++ {
		go listen()
	}
}
