package datastructs

type Item map[string]interface{}


func (i *Item) Valid() bool {
	return i != nil
}


type Data interface {
	Valid() bool
}
