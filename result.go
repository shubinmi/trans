package trans

type Result interface {
	Scan(dto interface{}) error
	Dto() interface{}
}
