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

package main_test

import (
	"testing"

	SpringCast "github.com/go-spring/spring-base/cast"
	"github.com/spf13/cast"
)

func BenchmarkToBool(b *testing.B) {

	b.Run("bool", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = SpringCast.ToBoolE(true)
		}
		for i := 0; i < b.N; i++ {
			_, _ = cast.ToBoolE(true)
		}
	})

	b.Run("bool ptr", func(b *testing.B) {
		v := true
		for i := 0; i < b.N; i++ {
			_, _ = SpringCast.ToBoolE(&v)
		}
		for i := 0; i < b.N; i++ {
			_, _ = cast.ToBoolE(&v)
		}
	})

	b.Run("string", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, _ = SpringCast.ToBoolE("true")
		}
		for i := 0; i < b.N; i++ {
			_, _ = cast.ToBoolE("true")
		}
	})

	b.Run("string ptr", func(b *testing.B) {
		v := "true"
		for i := 0; i < b.N; i++ {
			_, _ = SpringCast.ToBoolE(&v)
		}
		for i := 0; i < b.N; i++ {
			_, _ = cast.ToBoolE(&v)
		}
	})

}

func BenchmarkToString(b *testing.B) {

}
