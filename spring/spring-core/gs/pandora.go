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

package gs

import (
	"errors"
	"reflect"

	"github.com/go-spring/spring-core/arg"
	"github.com/go-spring/spring-core/bean"
	"github.com/go-spring/spring-core/conf"
	"github.com/go-spring/spring-core/log"
	"github.com/go-spring/spring-core/util"
)

// Pandora 请谨慎使用该接口提供的方法。
type Pandora interface {
	Prop(key string, opts ...conf.GetOption) interface{}
	Get(i interface{}, opts ...GetOption) error
	Find(selector bean.Selector) ([]bean.Definition, error)
	Collect(i interface{}, selectors ...bean.Selector) error
	Bind(i interface{}, opts ...conf.BindOption) error
	Wire(objOrCtor interface{}, ctorArgs ...arg.Arg) (interface{}, error)
	Go(fn interface{}, args ...arg.Arg)
	Invoke(fn interface{}, args ...arg.Arg) ([]interface{}, error)
}

type pandora struct {
	c *Container
}

// Prop 返回 key 转为小写后精确匹配的属性值，不存在返回 nil。
func (p *pandora) Prop(key string, opts ...conf.GetOption) interface{} {
	return p.c.p.Get(key, opts...)
}

type getArg struct {
	selector bean.Selector
}

type GetOption func(arg *getArg)

func Use(s bean.Selector) GetOption {
	return func(arg *getArg) {
		arg.selector = s
	}
}

// Get 获取单例 bean，若多于 1 个则 panic；找到返回 true 否则返回 false。
// 它和 FindBean 的区别是它在调用后能够保证返回的 bean 已经完成了注入和绑定过程。
func (p *pandora) Get(i interface{}, opts ...GetOption) error {

	if i == nil {
		return errors.New("i can't be nil")
	}

	p.c.callAfterRefreshing()

	// 使用指针才能够对外赋值
	if reflect.TypeOf(i).Kind() != reflect.Ptr {
		return errors.New("i must be pointer")
	}

	a := getArg{selector: bean.Selector("")}
	for _, opt := range opts {
		opt(&a)
	}

	w := toAssembly(p.c)
	v := reflect.ValueOf(i).Elem()
	return w.getBean(toSingletonTag(a.selector), v)
}

func (p *pandora) Find(selector bean.Selector) ([]bean.Definition, error) {
	return p.c.find(selector)
}

// Collect 收集数组或指针定义的所有符合条件的 bean，收集到返回 true，否则返
// 回 false。该函数有两种模式:自动模式和指定模式。自动模式是指 selectors 参数为空，
// 这时候不仅会收集符合条件的单例 bean，还会收集符合条件的数组 bean (是指数组的元素
// 符合条件，然后把数组元素拆开一个个放到收集结果里面)。指定模式是指 selectors 参数
// 不为空，这时候只会收集单例 bean，而且要求这些单例 bean 不仅需要满足收集条件，而且
// 必须满足 selector 条件。另外，自动模式下不对收集结果进行排序，指定模式下根据
// selectors 列表的顺序对收集结果进行排序。
func (p *pandora) Collect(i interface{}, selectors ...bean.Selector) error {
	p.c.callAfterRefreshing()

	v := reflect.ValueOf(i)
	if v.Kind() != reflect.Ptr {
		return errors.New("i must be slice ptr")
	}

	var tag collectionTag
	for _, selector := range selectors {
		s := toSingletonTag(selector)
		tag.beanTags = append(tag.beanTags, s)
	}
	return toAssembly(p.c).collectBeans(tag, v.Elem())
}

// Bind 绑定结构体属性。
func (p *pandora) Bind(i interface{}, opts ...conf.BindOption) error {
	return p.c.p.Bind(i, opts...)
}

// Wire 对对象或者构造函数的结果进行依赖注入和属性绑定，返回处理后的对象
func (p *pandora) Wire(objOrCtor interface{}, ctorArgs ...arg.Arg) (interface{}, error) {
	p.c.callAfterRefreshing()
	assembly := toAssembly(p.c)
	b := NewBean(objOrCtor, ctorArgs...)
	// 这里使用 wireBean 是为了追踪 bean 的注入路径。
	if err := assembly.wireBean(b); err != nil {
		return nil, err
	}
	return b.Interface(), nil
}

// Go 安全地启动一个 goroutine
func (p *pandora) Go(fn interface{}, args ...arg.Arg) {
	p.c.callAfterRefreshing()

	fnType := reflect.TypeOf(fn)
	if !util.IsFuncType(fnType) || !util.ReturnNothing(fnType) {
		panic(errors.New("fn should be func()"))
	}

	r := arg.Bind(fn, args, arg.Skip(1))

	p.c.wg.Add(1)
	go func() {
		defer p.c.wg.Done()

		defer func() {
			if r := recover(); r != nil {
				log.Error(r)
			}
		}()

		_, err := r.Call(toAssembly(p.c))
		if err != nil {
			log.Error(err.Error())
		}
	}()
}

// Invoke 立即执行一个一次性的任务
func (p *pandora) Invoke(fn interface{}, args ...arg.Arg) ([]interface{}, error) {
	p.c.callAfterRefreshing()
	if fnType := reflect.TypeOf(fn); util.IsFuncType(fnType) {
		if util.ReturnNothing(fnType) || util.ReturnOnlyError(fnType) {
			r := arg.Bind(fn, args, arg.Skip(1))
			callResult, err := r.Call(toAssembly(p.c))
			if err != nil {
				return nil, err
			}
			var arr []interface{}
			for _, v := range callResult {
				arr = append(arr, v.Interface())
			}
			return arr, nil
		}
	}
	return nil, errors.New("fn should be func() or func()error")
}