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

package push

import (
	"reflect"
	"testing"
	"time"

	pb "go.opentelemetry.io/contrib/sdk/dynamicconfig/internal/proto/experimental/metrics/configservice"
	controllerTest "go.opentelemetry.io/otel/sdk/metric/controller/test"
)

func makeConfig() *pb.MetricConfigResponse {
	oneSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "one",
				},
			},
		},
		PeriodSec: 21,
	}
	twoSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "two",
				},
			},
		},
		PeriodSec: 42,
	}
	redSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "red",
				},
			},
		},
		PeriodSec: 49,
	}
	blueSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "blue",
				},
			},
		},
		PeriodSec: 0,
	}

	config := pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			&oneSchedule,
			&twoSchedule,
			&redSchedule,
			&blueSchedule,
		},
	}

	return &config
}

func makeBadConfig() *pb.MetricConfigResponse {
	oneSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "one",
				},
			},
		},
		PeriodSec: -1,
	}
	config := pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			&oneSchedule,
		},
	}

	return &config
}

func makeZeroConfig() *pb.MetricConfigResponse {
	oneSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "one",
				},
			},
		},
		PeriodSec: 0,
	}
	twoSchedule := pb.MetricConfigResponse_Schedule{
		InclusionPatterns: []*pb.MetricConfigResponse_Schedule_Pattern{
			{
				Match: &pb.MetricConfigResponse_Schedule_Pattern_StartsWith{
					StartsWith: "two",
				},
			},
		},
		PeriodSec: 0,
	}
	config := pb.MetricConfigResponse{
		Schedules: []*pb.MetricConfigResponse_Schedule{
			&oneSchedule,
			&twoSchedule,
		},
	}

	return &config
}

func TestApplySchedules(t *testing.T) {
	config := makeConfig()
	matcher := PeriodMatcher{}

	newPeriod, err := matcher.ApplySchedules(config.Schedules)
	if err != nil {
		t.Errorf("fail to apply schedules: %v", err)
	}

	if !reflect.DeepEqual(config.Schedules, matcher.sched) {
		t.Errorf("consumed schedule does not match in memory version")
	}

	if len(matcher.metrics) != 0 {
		t.Errorf("metrics map not reset")
	}

	if newPeriod != 7*time.Second {
		t.Errorf("expected export period to be 7s, got: %v", newPeriod)
	}
}

func TestApplySchedulesBad(t *testing.T) {
	config := makeBadConfig()
	matcher := PeriodMatcher{}

	_, err := matcher.ApplySchedules(config.Schedules)
	if err == nil {
		t.Errorf("expected schedules to throw error: %v", config.Schedules)
	}

	config.Schedules = nil
	_, err = matcher.ApplySchedules(config.Schedules)
	if err == nil {
		t.Errorf("expected error with empty schedules")
	}
}

func TestApplySchedulesZero(t *testing.T) {
	config := makeZeroConfig()
	matcher := PeriodMatcher{}

	newPeriod, err := matcher.ApplySchedules(config.Schedules)
	if err != nil {
		t.Errorf("fail to apply schedules: %v", config.Schedules)
	}

	if newPeriod != 0 {
		t.Errorf("expected period=0, got: %v", newPeriod)
	}
}

func TestGCD(t *testing.T) {
	cases := map[[2]int32]int32{
		{5, 10}:           5,
		{1223456, 654355}: 13,
		{12, 5}:           1,
		{40, 0}:           40,
		{0, 40}:           40,
		{7, 7}:            7,
	}

	for input, result := range cases {
		computation := gcd(input[0], input[1])
		if computation != result {
			t.Errorf("expected gcd(%d, %d) = %d, got %d", input[0], input[1], result, computation)
		}
	}
}

func TestGCDPanic(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("gcd(0, 0) did not panic")
		}
	}()

	gcd(0, 0)
}

func TestGetExportPeriod(t *testing.T) {
	config := makeConfig()
	exportPeriod := getExportPeriod(config.Schedules)
	if exportPeriod != 7*time.Second {
		t.Errorf("expected export period to be 7s, got: %v", exportPeriod)
	}
}

func TestGetExportPeriodPanic(t *testing.T) {
	defer func() {
		if err := recover(); err == nil {
			t.Errorf("matcher did not consume schedules, but did not panic")
		}
	}()

	getExportPeriod([]*pb.MetricConfigResponse_Schedule{})
}

func TestBuildRule(t *testing.T) {
	mockClock := controllerTest.NewMockClock()
	matcher := PeriodMatcher{}
	matcher.MarkStart(mockClock.Now())

	config := makeConfig()
	_, err := matcher.ApplySchedules(config.Schedules)
	if err != nil {
		t.Errorf("fail to apply schedules: %v", err)
	}

	mockClock.Add(7 * time.Second)
	rule := matcher.BuildRule(mockClock.Now())
	if rule("one-fish") || rule("two-fish") || rule("red-fish") || rule("blue-fish") {
		t.Errorf("no schedule should match at time=7")
	}

	mockClock.Add(14 * time.Second)
	rule = matcher.BuildRule(mockClock.Now())
	if !rule("one-fish") || rule("two-fish") || rule("red-fish") || rule("blue-fish") {
		t.Errorf("only one* schedule should match at time=21")
	}

	mockClock.Add(21 * time.Second)
	rule = matcher.BuildRule(mockClock.Now())
	if !rule("one-fish") || !rule("two-fish") || rule("red-fish") || rule("blue-fish") {
		t.Errorf("only one* and two* schedules should match at time=42")
	}

	mockClock.Add(7 * time.Second)
	rule = matcher.BuildRule(mockClock.Now())
	if rule("one-fish") || rule("two-fish") || !rule("red-fish") || rule("blue-fish") {
		t.Errorf("only red* schedule should match at time=49")
	}

	mockClock.Add(245 * time.Second)
	rule = matcher.BuildRule(mockClock.Now())
	if !rule("one-fish") || !rule("two-fish") || !rule("red-fish") || rule("blue-fish") {
		t.Errorf("one*, two*, and red* schedules should match at time=294")
	}
}
