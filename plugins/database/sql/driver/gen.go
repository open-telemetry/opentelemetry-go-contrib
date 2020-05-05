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

package driver

//go:generate wrappergen -basetype=driver.Driver -exttypes=driver.DriverContext -extrafields=setup,*tracingSetup -prefix=traceDD -newfuncname=newDriver

//go:generate wrappergen -basetype=driver.Conn -exttypes=driver.ConnBeginTx;driver.ConnPrepareContext;driver.Execer;driver.ExecerContext;driver.NamedValueChecker;driver.Pinger;driver.Queryer;driver.QueryerContext;driver.SessionResetter -extrafields=setup,*tracingSetup -prefix=traceDC -newfuncname=newConn

//go:generate wrappergen -basetype=driver.Stmt -exttypes=driver.ColumnConverter;driver.NamedValueChecker;driver.StmtExecContext;driver.StmtQueryContext -extrafields=setup,*tracingSetup;savedQuery,string -prefix=traceDS -newfuncname=newStmt
