package tests

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"kubelilin/domain/business/tenant"
	"strconv"
	"testing"
)

func TestRoleMenuList(t *testing.T) {
	dsn := "root:1234abcd@tcp(cdb-amqub3mo.bj.tencentcdb.com:10042)/sgr_paas?charset=utf8&parseTime=True"
	db1, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	service := tenant.NewSysMenuService(db1)

	dd := service.GetRoleMenuIdList(1)

	assert.Equal(t, len(dd) > 0, true)
}

func TestSlices(t *testing.T) {
	slice := make([]int, 100)
	p := &slice

	slice[0] = 111
	slice[1] = 222

	fmt.Println(slice[0])
	fmt.Println(slice[1])
	fmt.Println((*p)[0])
	fmt.Println((*p)[1])

}

func TestPointer(t *testing.T) {
	name := "deployment"
	var ss *string
	ss = &name

	println(*ss)

}

func TestConvert(t *testing.T) {
	var a int
	_ = StringToNumber("2", &a)
	assert.Equal(t, a == 2, true)

	var b int64
	_ = StringToNumber("22", &b)
	assert.Equal(t, b == 22, true)

	var c float64
	_ = StringToNumber("22.22", &c)
	assert.Equal(t, c == 22.22, true)

	PrintAnyValue(1)
}

func PrintAnyValue[T any](v T) {
	fmt.Printf("%v", v)
}

type Number interface {
	int | int32 | int64 | uint32 | uint64 | float64
}

func StringToNumber[N Number](strNumber string, outNumber *N) error {
	switch (interface{})(*outNumber).(type) {
	case int:
		cn, err := strconv.Atoi(strNumber)
		*outNumber = N(cn)
		return err
	case int32:
		cn, err := strconv.ParseInt(strNumber, 10, 32)
		*outNumber = N(cn)
		return err
	case int64:
		cn, err := strconv.ParseInt(strNumber, 10, 64)
		*outNumber = N(cn)
		return err
	case uint32:
		cn, err := strconv.ParseUint(strNumber, 10, 32)
		*outNumber = N(cn)
		return err
	case uint64:
		cn, err := strconv.ParseUint(strNumber, 10, 64)
		*outNumber = N(cn)
		return err
	case float64:
		cn, err := strconv.ParseFloat(strNumber, 64)
		*outNumber = N(cn)
		return err
	}
	return nil
}
