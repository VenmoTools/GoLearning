package conf

import (
	"testing"
)

var data = []string{"t", "T", "True", "1", "ok", "y"}
var fake = []interface{}{1, false, "True", 0}

func TestConv_GetAsBool(t *testing.T) {
	for _, d := range data {
		c := conv{d}
		res, err := c.GetAsBool()
		if err != nil {
			t.Error(err)
		}
		t.Log("convert ", d, "result", res)
	}

	defer func() {
		err := recover()
		if err != nil {
			t.Error(err)
		}
	}()

	for _, d := range fake {
		c := conv{d}

		res, err := c.GetAsBool()
		if err != nil {
			t.Error(err)
		}
		t.Log("convert ", d, "result", res)
	}

}

var strData = []string{"3.16", "0.0", "0", "-1", "123123", "y"}
var strFake = []interface{}{1, false, "True", 0}

func TestConv_GetAsFloat32(t *testing.T) {
	for _, d := range strData {
		c := conv{d}
		res, err := c.GetAsFloat32()
		if err != nil {
			t.Error(err)
		}
		t.Log("convert ", d, "result", res)

	}
	defer func() {
		err := recover()
		if err != nil {
			t.Error(err)
		}
	}()

	for _, d := range strFake {
		c := conv{d}

		res, err := c.GetAsFloat32()
		if err != nil {
			t.Error(err)
		}
		t.Log("convert ", d, "result", res)

	}
}

func TestConv_GetAsFloat64(t *testing.T) {
	for _, d := range strData {
		c := conv{d}
		res, err := c.GetAsFloat64()
		if err != nil {
			t.Error(err)
		}
		t.Log("convert ", d, "result", res)

	}
	defer func() {
		err := recover()
		if err != nil {
			t.Error(err)
		}
	}()

	for _, d := range strFake {
		c := conv{d}

		res, err := c.GetAsFloat64()
		if err != nil {
			t.Log(err)
		}
		t.Log("convert ", d, "result", res)
	}
}

func TestConv_GetAsInt(t *testing.T) {

}

func TestConv_GetAsString(t *testing.T) {

}

func TestValue_Get(t *testing.T) {
	v := Value{}
	v.Put("num",1)
	v.Put("str","123")
	v.Put("float",3.14)


	t.Log(v.Get("num").GetAsInt())
	t.Log(v.Get("str").GetAsString())
	t.Log(v.Get("float").GetAsFloat32())


}

func TestValue_Put(t *testing.T) {
	v := Value{}
	v.Put("num",1)
	v.Put("str","123")
	v.Put("float",3.14)

	if v.cap != 3 {
		t.Error("cap error")
	}
}
