package crud

// Store is a simplified interface to a key-blob store supporting CRUD operations.
type Store interface {
	List() ([]string, error)
	Store(name string, data []byte) error
	Read(name string) ([]byte, error)
	Delete(name string) error
}
