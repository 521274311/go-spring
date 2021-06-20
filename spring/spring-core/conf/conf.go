/*
 * Copyright 2012-2019 the original author or authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package conf 提供读取属性列表的方法，并且通过扩展机制支持各种格式的属性文件。
package conf

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/go-spring/spring-core/contain"
	"github.com/spf13/cast"
)

const rootKey = "$"

// Properties 提供创建和读取属性列表的方法。它使用扁平的 map[string]string 结
// 构存储数据，属性的 key 可以是 a.b.c 或者 a[0].b 两种形式，a.b.c 表示从 map
// 结构中获取属性值，a[0].b 表示从切片结构中获取属性值，并且 key 是大小写敏感的。
type Properties struct{ m map[string]string }

// New 返回一个空的属性列表。
func New() *Properties {
	return &Properties{m: make(map[string]string)}
}

// Map 返回一个由 map 创建的属性列表。
func Map(m map[string]interface{}) *Properties {
	p := New()
	for k, v := range m {
		p.Set(k, v)
	}
	return p
}

// Load 返回一个由属性文件创建的属性列表，file 可以是绝对路径，也可以是相对路径。
func Load(file string) (*Properties, error) {
	p := New()
	if err := p.Load(file); err != nil {
		return nil, err
	}
	return p, nil
}

// Load 返回一个由属性文件创建的属性列表，file 可以是绝对路径，也可以是相对路径。
func (p *Properties) Load(file string) error {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}
	return p.Read(b, filepath.Ext(file))
}

// Read 返回一个由 []byte 创建的属性列表，ext 是文件扩展名，如 .yaml、.toml 等。
func Read(b []byte, ext string) (*Properties, error) {
	p := New()
	if err := p.Read(b, ext); err != nil {
		return nil, err
	}
	return p, nil
}

// Read 返回一个由 []byte 创建的属性列表，ext 是文件扩展名，如 .yaml、.toml 等。
func (p *Properties) Read(b []byte, ext string) error {

	r, ok := readers[ext]
	if !ok {
		return fmt.Errorf("unsupported file type %s", ext)
	}

	m, err := r(b)
	if err != nil {
		return err
	}

	for k, v := range m {
		p.Set(k, v)
	}
	return nil
}

// Keys 返回属性 key 的列表。
func (p *Properties) Keys() []string {
	keys := make([]string, 0, len(p.m))
	for k := range p.m {
		keys = append(keys, k)
	}
	return keys
}

type getArg struct {
	def interface{} // 默认值
}

type GetOption func(arg *getArg)

// Def 为 Get 方法设置默认值。
func Def(v interface{}) GetOption {
	return func(arg *getArg) {
		arg.def = v
	}
}

// Get 获取 key 对应的属性值，注意 key 是大小写敏感的。当 key 对应的属性值存在时，
// 或者 key 对应的属性值不存在但设置了默认值时，Get 方法返回 string 类型的数据，
// 当 key 对应的属性值不存在且没有设置默认值时 Get 方法返回 nil。因此可以通过判断
// Get 方法的返回值是否为 nil 来判断 key 对应的属性值是否存在。
func (p *Properties) Get(key string, opts ...GetOption) interface{} {

	key = strings.TrimPrefix(key, rootKey+".")
	if val, ok := p.m[key]; ok {
		return val
	}

	arg := getArg{}
	for _, opt := range opts {
		opt(&arg)
	}

	if arg.def != nil {
		return cast.ToString(arg.def)
	}
	return nil
}

// Set 设置 key 对应的属性值，如果 key 对应的属性值已经存在则 Set 方法会覆盖旧
// 值。Set 方法除了支持 string 类型的属性值，还支持 int、uint、bool 等其他基础
// 数据类型的属性值。特殊情况下，Set 方法也支持 slice 、map 与基础数据类型组合构
// 成的属性值，其处理方式是将组合结构层层展开，可以将组合结构看成一棵树，那么叶子结
// 点的路径就是属性的 key，叶子结点的值就是属性的值。
func (p *Properties) Set(key string, val interface{}) {
	switch v := reflect.ValueOf(val); v.Kind() {
	case reflect.Map:
		for _, k := range v.MapKeys() {
			mapValue := v.MapIndex(k).Interface()
			mapKey := cast.ToString(k.Interface())
			p.Set(key+"."+mapKey, mapValue)
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			subKey := fmt.Sprintf("%s[%d]", key, i)
			subValue := v.Index(i).Interface()
			p.Set(subKey, subValue)
		}
	default:
		p.m[key] = cast.ToString(val)
	}
}

type bindArg struct {
	tag string
}

type BindOption func(arg *bindArg)

// Key 设置绑定使用的 key 。
func Key(key string) BindOption {
	return func(arg *bindArg) {
		arg.tag = "${" + key + "}"
	}
}

// Tag 设置绑定使用的 tag 。
func Tag(tag string) BindOption {
	return func(arg *bindArg) {
		arg.tag = tag
	}
}

// Bind 将 key 对应的属性值绑定到某个数据类型的实例上。i 必须是一个指针，只有这
// 样才能将修改传递出去。Bind 方法使用 tag 字符串对数据实例进行属性绑定，其语法
// 为 value:"${a:=b}"，其中 value 表示属性绑定，${} 表示属性引用，a 表示属性
// 的名称，:=b 表示为属性设置默认值。而且 tag 字符串还支持在默认值中进行嵌套引用
// ，即 ${a:=${b}}。当然，还有两点需要特别说明：
// 一是对 array、slice、map、struct 这些复合类型不能设置非空默认值，因为如果
// 默认值太长会影响阅读体验，而且解析起来也并不容易；
// 二是可以省略属性名而只有默认值，即 ${:=b}，原因是某些情况下属性名可能没想好或
// 者不太重要，比如，得益于字符串差值的实现，这种语法可以用于动态生成新的属性值，
// 也有人认为这是一种对 Golang 缺少默认值语法的补充，Bug is Feature。
func (p *Properties) Bind(i interface{}, opts ...BindOption) error {

	var v reflect.Value

	switch i.(type) {
	case reflect.Value:
		v = i.(reflect.Value)
	default:
		if v = reflect.ValueOf(i); v.Kind() != reflect.Ptr {
			return errors.New("属性绑定的对象必须是一个指针")
		}
		v = v.Elem()
	}

	arg := bindArg{}
	Key(rootKey)(&arg)

	for _, opt := range opts {
		opt(&arg)
	}

	t := v.Type()
	s := t.Name()
	if s == "" {
		switch t.Kind() {
		case reflect.Map, reflect.Slice, reflect.Array:
			s = t.Elem().Name()
			if s == "" {
				s = t.Elem().String()
			}
		default:
			s = t.String()
		}
	}

	return bind(p, v, arg.tag, bindOption{path: s})
}

// Group 对属性列表的 key 按照 prefix 作为前缀进行分组，然后返回分组的名称。
func (p *Properties) Group(prefix string) []string {

	trimPrefix := func(key, prefix string) (string, bool) {
		if prefix == rootKey {
			return key, true
		}
		if !strings.HasPrefix(key, prefix+".") {
			return "", false
		}
		return strings.TrimPrefix(key, prefix+"."), true
	}

	var groups []string
	for _, key := range p.Keys() {
		s, ok := trimPrefix(key, prefix)
		if !ok {
			continue
		}
		k := strings.Split(s, ".")[0]
		if contain.Strings(groups, k) >= 0 {
			continue
		}
		groups = append(groups, k)
	}
	return groups
}

// Resolve 解析字符串中包含的属性引用即 ${key:=def} 的内容，且支持递归引用。
func (p *Properties) Resolve(s string) (string, error) {
	return resolveString(p, s)
}