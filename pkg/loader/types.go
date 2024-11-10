package loader

type Camunda7Convertor interface {
	Convert(process string) (string, error)
	Check(process string) (bool, error)
}

type InternalFormat interface {
	Convert(process string) (string, error)
	Check(process string) (bool, error)
}
