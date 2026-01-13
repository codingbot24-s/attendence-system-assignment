package main

func main() {
	db := ConnectToDB()
	_ = NewDB(db)
}
