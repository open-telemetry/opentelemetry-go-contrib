// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package otelbeego

import (
	"github.com/astaxie/beego"
)

type ExampleController struct {
	beego.Controller
}

func (c *ExampleController) Get() {
	// name of the template in the views directory
	c.TplName = "index.tpl"

	// explicit call to Render
	if err := Render(&c.Controller); err != nil {
		c.Abort("500")
	}
}

func ExampleRender() {
	//  Init the trace and meter provider

	// Disable autorender
	beego.BConfig.WebConfig.AutoRender = false

	// Create routes
	beego.Router("/", &ExampleController{})

	// Create the middleware
	mware := NewOTelBeegoMiddleWare("exampe-server")

	// Start the server using the OTel middleware
	beego.RunWithMiddleWares(":7777", mware)
}
