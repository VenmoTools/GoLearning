package conf

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
)

type conv struct {
	data interface{}
}

func (c *conv) GetAsString() string {
	if c.data == nil {
		return ""
	}
	return c.data.(string)
}

func (c *conv) GetAsInt() int {
	if c.data == nil {
		return 0
	}
	return c.data.(int)
}

func (c *conv) GetAsBool() (bool, error) {
	if c.data == nil {
		return false, errors.New("the data is nil")
	}
	if d, ok := c.data.(string); ok {
		switch d {
		case "1", "t", "T", "true", "True", "ok", "OK", "Ok", "Y", "YES", "yes", "y":
			return true, nil
		default:
			res, err := strconv.ParseBool(d)
			if err != nil {
				return false, err
			}
			return res, nil

		}
	}
	if d, ok := c.data.(int); ok {
		return 0 == d, nil
	}
	if d, ok := c.data.(bool); ok {
		return d, nil
	}
	return false, errors.New(fmt.Sprintf("can not conver %#v as type bool", c.data))
}

func (c *conv) GetAsFloat32() (float32, error) {
	if c.data == nil {
		return 0, errors.New("the data is nil")
	}
	if d, ok := c.data.(float32); ok {
		return d, nil
	}
	if d, ok := c.data.(float64); ok {
		return float32(d), nil
	}
	if d, ok := c.data.(string); ok {
		res, err := strconv.ParseFloat(d, 10)
		if err != nil {
			return 0, err
		}
		return float32(res), nil
	}
	return 0, errors.New(fmt.Sprintf("can not conver %#v as type float32", c.data))
}

func (c *conv) GetAsFloat64() (float64, error) {
	if c.data == nil {
		return 0, errors.New("the data is nil")
	}
	if d, ok := c.data.(float64); ok {
		return d, nil
	}
	if d, ok := c.data.(float32); ok {
		return float64(d), nil
	}
	if d, ok := c.data.(string); ok {
		res, err := strconv.ParseFloat(d, 10)
		if err != nil {
			return 0, err
		}
		return res, nil
	}
	return 0, errors.New(fmt.Sprintf("can not conver %#v as type float32", c.data))
}

func (c *conv) GetAsFloat64Slice() ([]float64, error) {
	arr := make([]float64, 0)

	if c.data == nil {
		return arr, errors.New("the data is nil")
	}
	if d, ok := c.data.(string); ok {
		d = strings.TrimSpace(d[1 : len(d)-1])
		data := strings.Split(d, ",")
		for _, x := range data {
			res, err := strconv.ParseFloat(x, 10)
			if err != nil {
				continue
			}
			arr = append(arr, res)
		}
		return arr, nil
	}
	return arr, nil
}

func (c *conv) GetAsIntSlice() ([]int, error) {
	arr := make([]int, 0)

	if c.data == nil {
		return arr, errors.New("the data is nil")
	}
	if d, ok := c.data.(string); ok {
		d = strings.TrimSpace(d[1 : len(d)-1])
		data := strings.Split(d, ",")
		for _, x := range data {
			if res, err := strconv.ParseInt(x, 10, 32); err == nil {
				arr = append(arr, int(res))
			}
		}
		return arr, nil
	}
	return arr, nil
}

func (c *conv) GetAsStringSlice() ([]string, error) {
	arr := make([]string, 0)

	if c.data == nil {
		return arr, errors.New("the data is nil")
	}
	if d, ok := c.data.(string); ok {
		d = strings.TrimSpace(d[1 : len(d)-1])
		data := strings.Split(d, ",")
		for _, x := range data {
			arr = append(arr, x)
		}
		return arr, nil
	}
	return arr, nil
}

type Value struct {
	value map[string]interface{}
	cap   uint64
}

func NewValue() *Value {
	return &Value{
		value: make(map[string]interface{}),
		cap:   0,
	}
}

func (v *Value) Get(key string) (*conv, bool) {
	data, ok := v.value[key]
	return &conv{data: data}, ok
}

func (v *Value) Put(key string, value string) {
	v.value[key] = value
	atomic.AddUint64(&v.cap, 1)
}
